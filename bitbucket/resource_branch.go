package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/migara/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceBranch() *schema.Resource {
	return &schema.Resource{
		Create: resourceBranchCreate,
		Update: resourceBranchUpdate,
		Read:   resourceBranchRead,
		Delete: resourceBranchDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"owner": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repo_slug": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"branch_name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"target": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"default_merge_strategy": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"merge_strategies": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func newBranchFromResource(d *schema.ResourceData) *bitbucket.Branch {
	branch := &bitbucket.Branch{
		Name: d.Get("branch_name").(string),
		Target: &bitbucket.Commit{
			Hash: d.Get("target").(string),
		},
		// Type_:                d.Get("type").(string),
		// DefaultMergeStrategy: d.Get("default_merge_strategy").(string),
	}

	// if v, ok := d.GetOk("merge_strategies"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
	// 	branch.MergeStrategies = expandMergeStrategies(v.([]interface{}))
	// }

	return branch
}

func resourceBranchUpdate(d *schema.ResourceData, m interface{}) error {
	resourceBranchDelete(d, m)

	resourceBranchRead(d, m)
	return resourceBranchCreate(d, m)
}

func resourceBranchCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	refsApi := c.ApiClient.RefsApi
	branch := newBranchFromResource(d)

	owner := d.Get("owner").(string)
	repo_slug := d.Get("repo_slug").(string)
	branch_name := d.Get("branch_name").(string)

	_, _, err := refsApi.RepositoriesWorkspaceRepoSlugRefsBranchesPost(c.AuthContext, repo_slug, owner, *branch)
	if err != nil {
		return fmt.Errorf("error creating branch (%s): %w", branch, err)
	}

	d.SetId(string(fmt.Sprintf("%s:%s:%s", owner, repo_slug, branch_name)))

	return resourceBranchRead(d, m)
}

func resourceBranchRead(d *schema.ResourceData, m interface{}) error {
	id := d.Id()
	if id != "" {
		idparts := strings.Split(id, ":")
		if len(idparts) == 3 {
			d.Set("owner", idparts[0])
			d.Set("repo_slug", idparts[1])
			d.Set("branch_name", idparts[2])
		} else {
			return fmt.Errorf("incorrect ID format, should match `owner:repo_slug:branch_name`")
		}
	}

	owner := d.Get("owner").(string)
	repo_slug := d.Get("repo_slug").(string)
	branch_name := d.Get("branch_name").(string)

	c := m.(Clients).genClient
	refsApi := c.ApiClient.RefsApi

	branchRes, res, err := refsApi.RepositoriesWorkspaceRepoSlugRefsBranchesNameGet(c.AuthContext, branch_name, repo_slug, owner)

	if err != nil {
		return fmt.Errorf("error reading branch (%s): %w", d.Id(), err)
	}
	if res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Branch (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	saveStrategies(d, branchRes)
	return nil
}

func saveStrategies(d *schema.ResourceData, branchRes bitbucket.Branch) {
	var strategies []interface{}
	if len(branchRes.MergeStrategies) > 0 {
		strategies = make([]interface{}, 0, len(branchRes.MergeStrategies))
		for _, x := range branchRes.MergeStrategies {
			strategies = append(strategies, x)
		}
	}

	// var target map[string]interface{}
	// if len(branchRes.MergeStrategies) > 0 {
	// 	strategies = make([]interface{}, 0, len(branchRes.MergeStrategies))
	// 	for _, x := range branchRes.MergeStrategies {
	// 		strategies = append(sm, x)
	// 	}
	// }

	d.Set("type", branchRes.Type_)
	d.Set("links", branchRes.Links)
	d.Set("target", branchRes.Target)
	d.Set("default_merge_strategy", branchRes.DefaultMergeStrategy)
	d.Set("merge_strategies", strategies)
}

func resourceBranchDelete(d *schema.ResourceData, m interface{}) error {

	owner := d.Get("owner").(string)
	repo_slug := d.Get("repo_slug").(string)
	branch_name := d.Get("branch_name").(string)

	c := m.(Clients).genClient
	refsApi := c.ApiClient.RefsApi

	_, err := refsApi.RepositoriesWorkspaceRepoSlugRefsBranchesNameDelete(c.AuthContext, branch_name, repo_slug, owner)
	if err != nil {
		return fmt.Errorf("error deleting branch (%s): %w", d.Id(), err)
	}

	return nil
}

func expandMergeStrategies(s []interface{}) []string {
	if len(s) == 0 || s[0] == nil {
		return nil
	}

	strategies := make([]string, 0, len(s))

	for _, x := range s {
		strategies = append(strategies, x.(string))
	}

	return strategies
}

// func expandTarget(t interface{}) *bitbucket.Commit {

// 	commit := &bitbucket.Commit{}
// 	commit.Hash = t.(string)

// 	return commit
// }
