package bitbucket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceProject() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceProjectCreate,
		UpdateWithoutTimeout: resourceProjectUpdate,
		ReadWithoutTimeout:   resourceProjectRead,
		DeleteWithoutTimeout: resourceProjectDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"key": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"is_private": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"owner": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
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
				Optional: true,
				Computed: true,
				MaxItems: 1,
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

// type SmallProject struct {
// 	Links *bitbucket.ProjectLinks `json:"links,omitempty"`
// }

func newProjectFromResource(d *schema.ResourceData) *bitbucket.Project {
	project := &bitbucket.Project{
		Name:        d.Get("name").(string),
		IsPrivate:   d.Get("is_private").(bool),
		Description: d.Get("description").(string),
		Key:         d.Get("key").(string),
	}

	if v, ok := d.GetOk("link"); d.IsNewResource() && ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		project.Links = expandProjectLinks(v.([]interface{}))
	}

	return project
}

func resourceProjectUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	// client := m.(Clients).httpClient
	projectApi := c.ApiClient.ProjectsApi
	project := newProjectFromResource(d)

	var projectKey string
	projectKey = d.Get("key").(string)
	if projectKey == "" {
		//nolint:staticcheck
		projectKey = d.Get("key").(string)
	}

	log.Printf("[DEBUG] Project Update Body: %#v", project)
	project.Links = nil

	prj, res, err := projectApi.WorkspacesWorkspaceProjectsProjectKeyPut(c.AuthContext, *project, projectKey, d.Get("owner").(string))
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Project Update Res: %#v", prj)

	// if d.HasChange("link") {
	// 	if v, ok := d.GetOk("link"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {

	// 		smallProject := SmallProject{
	// 			Links: expandProjectLinks(v.([]interface{})),
	// 		}

	// 		payload, err := json.Marshal(smallProject)
	// 		if err != nil {
	// 			return diag.FromErr(err)
	// 		}

	// 		log.Printf("[DEBUG] Project Update Links Body: %s", string(payload))

	// 		_, err = client.Put(fmt.Sprintf("2.0/workspaces/%s/projects/%s",
	// 			d.Get("owner").(string), d.Get("key").(string),
	// 		), bytes.NewBuffer(payload))

	// 		if err != nil {
	// 			return diag.FromErr(err)
	// 		}

	// 	}
	// }

	return resourceProjectRead(ctx, d, m)
}

func resourceProjectCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	projectApi := c.ApiClient.ProjectsApi
	project := newProjectFromResource(d)

	owner := d.Get("owner").(string)

	log.Printf("[DEBUG] Project Create Body: %#v", project)

	projRes, res, err := projectApi.WorkspacesWorkspaceProjectsPost(c.AuthContext, *project, owner)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", owner, projRes.Key))

	return resourceProjectRead(ctx, d, m)
}

func resourceProjectRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	if id != "" {
		idparts := strings.Split(id, "/")
		if len(idparts) == 2 {
			d.Set("owner", idparts[0])
			d.Set("key", idparts[1])
		} else {
			return diag.Errorf("incorrect ID format, should match `owner/key`")
		}
	}

	var projectKey string
	projectKey = d.Get("key").(string)
	if projectKey == "" {
		projectKey = d.Get("key").(string)
	}

	c := m.(Clients).genClient
	projectApi := c.ApiClient.ProjectsApi

	projRes, res, err := projectApi.WorkspacesWorkspaceProjectsProjectKeyGet(c.AuthContext, projectKey, d.Get("owner").(string))

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Project (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("key", projRes.Key)
	d.Set("is_private", projRes.IsPrivate)
	d.Set("name", projRes.Name)
	d.Set("description", projRes.Description)
	d.Set("has_publicly_visible_repos", projRes.HasPubliclyVisibleRepos)
	d.Set("uuid", projRes.Uuid)
	d.Set("link", flattenProjectLinks(projRes.Links))

	return nil
}

func resourceProjectDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var projectKey string
	projectKey = d.Get("key").(string)
	if projectKey == "" {
		projectKey = d.Get("key").(string)
	}

	c := m.(Clients).genClient
	projectApi := c.ApiClient.ProjectsApi

	res, err := projectApi.WorkspacesWorkspaceProjectsProjectKeyDelete(c.AuthContext, projectKey, d.Get("owner").(string))
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func expandProjectLinks(l []interface{}) *bitbucket.ProjectLinks {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})

	if !ok {
		return nil
	}

	rp := &bitbucket.ProjectLinks{}

	if v, ok := tfMap["avatar"].([]interface{}); ok && len(v) > 0 {
		rp.Avatar = expandLink(v)
	}

	return rp
}

func flattenProjectLinks(rp *bitbucket.ProjectLinks) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"avatar": flattenLink(rp.Avatar),
	}

	return []interface{}{m}
}
