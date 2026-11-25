package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataProject() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadProject,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Description: "Datasource to retrieve project information",

		Schema: map[string]*schema.Schema{
			"enforced_signed_commits": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether signed commits are required for the project: true when the workspace is on a Premium plan and enforcement is on for the project or its workspace.",
			},
			"key": {
				Type:         schema.TypeString,
				Description:  "Project key",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},

			"workspace": {
				Type:         schema.TypeString,
				Description:  "Project workspace slug or {UUID}",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"is_private": {
				Type:        schema.TypeBool,
				Description: "Project is private",
				Optional:    true,
				Computed:    true,
			},
			"description": {
				Type:        schema.TypeString,
				Description: "Project description",
				Optional:    true,
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Project name",
				Computed:    true,
			},
			"owner": {
				Type:        schema.TypeList,
				Description: "Project owner information",
				Computed:    true,
				Optional:    true,
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
			"has_publicly_visible_repos": {
				Type:        schema.TypeBool,
				Description: "Repositories are publicly visible",
				Computed:    true,
			},
			"uuid": {
				Type:        schema.TypeString,
				Description: "Project UUID",
				Computed:    true,
			},
			"link": {
				Type:        schema.TypeList,
				Description: "Link information",
				Optional:    true,
				Computed:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"avatar": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"href": {
										Type:     schema.TypeString,
										Optional: true,
										DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
											return strings.HasPrefix(old, "https://bitbucket.org/account/user")
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Flattens the owner account info
func flattenTeam(o *bitbucket.Team) []interface{} {
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

func dataReadProject(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	projectKey := d.Get("key").(string)
	workspace := d.Get("workspace").(string)

	log.Printf("[DEBUG]: params for %s: %v", "dataReadProject", dumpResourceData(d, dataProject().Schema))

	url := fmt.Sprintf("2.0/workspaces/%s/projects/%s",
		workspace,
		projectKey,
	)

	client := m.(Clients).httpClient
	res, err := client.Get(url)
	if err != nil {
		return diag.FromErr(err)
	}

	if res.Body == nil {
		return diag.Errorf("error reading project information with params (%s): ", dumpResourceData(d, dataProject().Schema))
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

	var project bitbucket.Project
	decodeerr := json.Unmarshal(repoBody, &project)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	d.SetId(fmt.Sprintf("%s/%s", workspace, projectKey))

	d.Set("owner", flattenTeam(project.Owner))
	d.Set("key", project.Key)
	d.Set("is_private", project.IsPrivate)
	d.Set("name", project.Name)
	d.Set("description", project.Description)
	d.Set("has_publicly_visible_repos", project.HasPubliclyVisibleRepos)
	d.Set("uuid", project.Uuid)
	d.Set("enforced_signed_commits", project.EnforcedSignedCommits)
	d.Set("link", flattenProjectLinks(project.Links))

	return nil
}
