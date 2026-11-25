package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataRepository() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadRepository,
		Description:        "Datasource to retrieve repository information",
		Schema: map[string]*schema.Schema{
			"enforced_signed_commits": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether signed commits are required for the repository: true when the workspace is on a Premium plan and enforcement is on for the repository, its project or its workspace.",
			},
			"workspace": {
				Type:         schema.TypeString,
				Description:  "Workspace slug or UUID",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repo_slug": {
				Type:         schema.TypeString,
				Description:  "Repository slug or UUID",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"scm": {
				Type:        schema.TypeString,
				Description: "Repository SCM",
				Computed:    true,
			},
			"has_wiki": {
				Type:        schema.TypeBool,
				Description: "Repository has a Confluence page",
				Computed:    true,
			},
			"has_issues": {
				Type:        schema.TypeBool,
				Description: "If repository currently has JIRA issues assigned to it",
				Computed:    true,
			},
			"is_private": {
				Type:        schema.TypeBool,
				Description: "If repository is private",
				Optional:    true,
				Computed:    true,
			},
			"fork_policy": {
				Type:        schema.TypeString,
				Description: "Repository fork policy",
				Computed:    true,
			},
			"full_name": {
				Type:        schema.TypeString,
				Description: "Repository full name",
				Computed:    true,
			},
			"language": {
				Type:        schema.TypeString,
				Description: "Repository language",
				Computed:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "Repository description",
				Computed:    true,
			},
			"owner": {
				Type:        schema.TypeList,
				Description: "Repository owner information",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"username": {
							Type:        schema.TypeString,
							Description: "Owner username",
							Computed:    true,
						},
						"display_name": {
							Type:        schema.TypeString,
							Description: "Owner display name",
							Computed:    true,
						},
						"uuid": {
							Type:        schema.TypeString,
							Description: "Owner UUID",
							Computed:    true,
						},
					},
				},
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Repository name",
				Computed:    true,
			},
			"uuid": {
				Type:        schema.TypeString,
				Description: "Repository UUID",
				Computed:    true,
			},
			"main_branch": {
				Type:        schema.TypeString,
				Description: "Main branch name",
				Computed:    true,
			},
			"link": {
				Type:     schema.TypeList,
				Computed: true,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"avatar": {
							Type:        schema.TypeList,
							Description: "Link to avatar",
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"href": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"project": {
				Type:        schema.TypeList,
				Description: "Project information",
				Computed:    true,
				Optional:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Description: "Project name",
							Computed:    true,
						},
						"description": {
							Type:        schema.TypeString,
							Description: "Project description",
							Computed:    true,
						},
						"is_private": {
							Type:        schema.TypeBool,
							Description: "If project is private",
							Computed:    true,
						},
						"key": {
							Type:        schema.TypeString,
							Description: "Project key",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// Flattens the project info
func flattenProject(p *bitbucket.Project) []interface{} {
	if p == nil {
		return nil
	}
	return []interface{}{
		map[string]interface{}{
			"name":        p.Name,
			"is_private":  p.IsPrivate,
			"description": p.Description,
			"key":         p.Key,
		},
	}

}

// Flattens the owner account info
func flattenAccount(o *bitbucket.Account) []interface{} {
	if o == nil {
		return nil
	}
	return []interface{}{
		map[string]interface{}{
			"username":     o.Username,
			"display_name": o.DisplayName,
			"uuid":         o.Uuid,
		},
	}

}

// Flattens the repository info
func flattenRepository(r *bitbucket.Repository, d *schema.ResourceData) {
	if r == nil {
		return
	}

	d.Set("name", r.Name)
	d.Set("full_name", r.FullName)
	d.Set("language", r.Language)
	d.Set("owner", flattenAccount(r.Owner))
	d.Set("is_private", r.IsPrivate)
	d.Set("description", r.Description)
	d.Set("fork_policy", r.ForkPolicy)
	d.Set("has_wiki", r.HasWiki)
	d.Set("has_issues", r.HasIssues)
	d.Set("scm", r.Scm)
	d.Set("uuid", r.Uuid)
	d.Set("enforced_signed_commits", r.EnforcedSignedCommits)
	if r.Mainbranch != nil {
		d.Set("main_branch", r.Mainbranch.Name)
	}
	d.Set("project", flattenProject(r.Project))
	d.Set("link", flattenLinks(r.Links))
}

func dataReadRepository(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	repoSlug := d.Get("repo_slug").(string)
	workspace := d.Get("workspace").(string)

	log.Printf("[DEBUG]: params for %s: %v", "dataReadRepository", dumpResourceData(d, dataRepository().Schema))

	url := fmt.Sprintf("2.0/repositories/%s/%s",
		workspace,
		repoSlug,
	)

	client := m.(Clients).httpClient
	res, err := client.Get(url)
	if err != nil {
		return diag.FromErr(err)
	}
	if res == nil {
		return diag.Errorf("no response returned from repositories call. Make sure your credentials are accurate.")
	}

	if res.StatusCode == http.StatusNotFound {
		return diag.Errorf("unable to locate repository with slug/UUID %s in workspace %s", repoSlug, workspace)
	}

	if res.Body == nil {
		return diag.Errorf("error reading repo information with params (%s): ", dumpResourceData(d, dataRepository().Schema))
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	repoBody, readerr := io.ReadAll(res.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}
	log.Printf("[DEBUG] http response: %v", res)
	log.Printf("[DEBUG] repo response: %v", repoBody)

	var repo bitbucket.Repository
	decodeerr := json.Unmarshal(repoBody, &repo)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	d.SetId(repo.Uuid)
	flattenRepository(&repo, d)
	return nil
}
