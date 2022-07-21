package bitbucket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataProject() *schema.Resource {
	return &schema.Resource{
		Read: dataReadProject,

		Schema: map[string]*schema.Schema{
			"key": {
				Type:     schema.TypeString,
				Required: true,
				// ValidateFunc: validation.StringIsNotEmpty,
			},
			"owner": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"is_private": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"has_publicly_visible_repos": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"link": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"avatar": {
							Type:     schema.TypeList,
							Computed: true,
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
		},
	}
}

func dataReadProject(d *schema.ResourceData, m interface{}) error {

	projectKey := d.Get("key").(string)
	owner := d.Get("owner").(string)

	c := m.(Clients).genClient
	projectApi := c.ApiClient.ProjectsApi

	projRes, res, err := projectApi.WorkspacesWorkspaceProjectsProjectKeyGet(c.AuthContext, projectKey, owner)

	if err != nil {
		return fmt.Errorf("error reading project (%s): %w", d.Id(), err)
	}
	if res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Project (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("key", projectKey)
	d.Set("is_private", projRes.IsPrivate)
	d.Set("name", projRes.Name)
	d.Set("description", projRes.Description)
	d.Set("has_publicly_visible_repos", projRes.HasPubliclyVisibleRepos)
	d.Set("uuid", projRes.Uuid)
	d.Set("link", flattenProjectLinks(projRes.Links))

	d.SetId(string(fmt.Sprintf("%s/%s", owner, projectKey)))
	return nil
}
