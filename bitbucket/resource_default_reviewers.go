package bitbucket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDefaultReviewers() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceDefaultReviewersCreate,
		ReadWithoutTimeout:   resourceDefaultReviewersRead,
		UpdateWithoutTimeout: resourceDefaultReviewersUpdate,
		DeleteWithoutTimeout: resourceDefaultReviewersDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"owner": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"reviewers": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
			},
		},
	}
}

func resourceDefaultReviewersCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	prApi := c.ApiClient.PullrequestsApi

	repo := d.Get("repository").(string)
	workspace := d.Get("owner").(string)
	for _, user := range d.Get("reviewers").(*schema.Set).List() {
		userName := user.(string)

		_, _, err := prApi.RepositoriesWorkspaceRepoSlugDefaultReviewersTargetUsernamePut(c.AuthContext, repo, userName, workspace)
		if err := handleClientError(err); err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(fmt.Sprintf("%s/%s/reviewers", workspace, repo))
	return resourceDefaultReviewersRead(ctx, d, m)
}

func resourceDefaultReviewersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	prApi := c.ApiClient.PullrequestsApi

	options := bitbucket.PullrequestsApiRepositoriesWorkspaceRepoSlugDefaultReviewersGetOpts{}

	owner, repo, err := defaultReviewersId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var terraformReviewers []string

	for {
		reviewers, res, err := prApi.RepositoriesWorkspaceRepoSlugDefaultReviewersGet(c.AuthContext, repo, owner, &options)
		if err := handleClientError(err); err != nil {
			return diag.FromErr(err)
		}

		if res.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Default Reviewers (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		for _, reviewer := range reviewers.Values {
			terraformReviewers = append(terraformReviewers, reviewer.Uuid)
		}

		if reviewers.Next != "" {
			nextPage := reviewers.Page + 1
			options.Page = optional.NewInt32(nextPage)
		} else {
			break
		}
	}

	d.Set("owner", owner)
	d.Set("repository", repo)
	d.Set("reviewers", terraformReviewers)

	return nil
}

func resourceDefaultReviewersUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient

	prApi := c.ApiClient.PullrequestsApi
	oraw, nraw := d.GetChange("reviewers")
	o := oraw.(*schema.Set)
	n := nraw.(*schema.Set)

	add := n.Difference(o)
	remove := o.Difference(n)
	repo := d.Get("repository").(string)
	workspace := d.Get("owner").(string)

	for _, user := range add.List() {
		userName := user.(string)
		_, _, err := prApi.RepositoriesWorkspaceRepoSlugDefaultReviewersTargetUsernamePut(c.AuthContext, repo, userName, workspace)
		if err := handleClientError(err); err != nil {
			return diag.FromErr(err)
		}
	}

	for _, user := range remove.List() {
		userName := user.(string)
		_, err := prApi.RepositoriesWorkspaceRepoSlugDefaultReviewersTargetUsernameDelete(c.AuthContext, repo, userName, workspace)
		if err := handleClientError(err); err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceDefaultReviewersRead(ctx, d, m)
}

func resourceDefaultReviewersDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	prApi := c.ApiClient.PullrequestsApi

	repo := d.Get("repository").(string)
	workspace := d.Get("owner").(string)
	for _, user := range d.Get("reviewers").(*schema.Set).List() {
		userName := user.(string)
		_, err := prApi.RepositoriesWorkspaceRepoSlugDefaultReviewersTargetUsernameDelete(c.AuthContext, repo, userName, workspace)
		if err := handleClientError(err); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

func defaultReviewersId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 3 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected OWNER/REPOSITORY/reviewers", id)
	}

	return parts[0], parts[1], nil
}
