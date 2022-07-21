package bitbucket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/migara/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataProjects() *schema.Resource {
	return &schema.Resource{
		Read: dataReadProjects,

		Schema: map[string]*schema.Schema{
			"owner": {
				Type:     schema.TypeString,
				Required: true,
			},
			"projects": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
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
						// "link": {
						// 	Type:     schema.TypeList,
						// 	Computed: true,
						// 	Elem: &schema.Resource{
						// 		Schema: map[string]*schema.Schema{
						// 			"avatar": {
						// 				Type:     schema.TypeList,
						// 				Computed: true,
						// 				Elem: &schema.Resource{
						// 					Schema: map[string]*schema.Schema{
						// 						"href": {
						// 							Type:     schema.TypeString,
						// 							Computed: true,
						// 						},
						// 					},
						// 				},
						// 			},
						// 		},
						// 	},
						// },
					},
				},
			},
		},
	}
}

func dataReadProjects(d *schema.ResourceData, m interface{}) error {

	owner := d.Get("owner").(string)

	c := m.(Clients).genClient
	workspaceApi := c.ApiClient.WorkspacesApi

	var next string
	var projects []bitbucket.Project

	for {
		projRes, res, err := workspaceApi.WorkspacesWorkspaceProjectsGet(c.AuthContext, owner, next)

		if err != nil {
			return fmt.Errorf("error reading projects (%s): %w", d.Id(), err)
		}

		if res.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] Project (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}

		projects = append(projects, projRes.Values...)
		next = projRes.Next
		if next == "" {
			break
		}

	}

	_projects := make([]interface{}, 0, len(projects))

	for _, x := range projects {
		_projects = append(_projects, map[string]interface{}{
			"key":                        x.Key,
			"is_private":                 x.IsPrivate,
			"name":                       x.Name,
			"description":                x.Description,
			"has_publicly_visible_repos": x.HasPubliclyVisibleRepos,
			"uuid":                       x.Uuid,
		})
	}

	d.Set("projects", _projects)
	d.SetId(owner)

	return nil
}
