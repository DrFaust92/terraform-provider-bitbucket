package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type ProjectGroupPermission struct {
	Permission string        `json:"permission"`
	Group      *ProjectGroup `json:"group,omitempty"`
}

type ProjectGroup struct {
	Name      string              `json:"name,omitempty"`
	Slug      string              `json:"slug,omitempty"`
	Workspace bitbucket.Workspace `json:"workspace,omitempty"`
}

func resourceProjectGroupPermission() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceProjectGroupPermissionPut,
		ReadWithoutTimeout:   resourceProjectGroupPermissionRead,
		UpdateWithoutTimeout: resourceProjectGroupPermissionPut,
		DeleteWithoutTimeout: resourceProjectGroupPermissionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project_key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"group_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"permission": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"admin", "write", "read", "create-repo"}, false),
			},
		},
	}
}

func createProjectGroupPermission(d *schema.ResourceData) *ProjectGroupPermission {

	permission := &ProjectGroupPermission{
		Permission: d.Get("permission").(string),
	}

	return permission
}

func resourceProjectGroupPermissionPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	permission := createProjectGroupPermission(d)

	payload, err := json.Marshal(permission)
	if err != nil {
		return diag.FromErr(err)
	}

	workspace := d.Get("workspace").(string)
	projectKey := d.Get("project_key").(string)
	groupSlug := d.Get("group_slug").(string)

	permissionReq, err := client.Put(fmt.Sprintf("2.0/workspaces/%s/projects/%s/permissions-config/groups/%s",
		workspace,
		projectKey,
		groupSlug,
	), bytes.NewBuffer(payload))

	if err != nil {
		return diag.FromErr(err)
	}

	body, readerr := io.ReadAll(permissionReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	decodeerr := json.Unmarshal(body, &permission)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	if d.IsNewResource() {
		d.SetId(fmt.Sprintf("%s:%s:%s", workspace, projectKey, groupSlug))
	}

	return resourceProjectGroupPermissionRead(ctx, d, m)
}

func resourceProjectGroupPermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, projectKey, groupSlug, err := projectGroupPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	permissionReq, err := client.Get(fmt.Sprintf("2.0/workspaces/%s/projects/%s/permissions-config/groups/%s",
		workspace,
		projectKey,
		groupSlug,
	))

	if permissionReq != nil && permissionReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Project Group Permission (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	var permission ProjectGroupPermission

	body, readerr := io.ReadAll(permissionReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("Project Group Permission raw is: %#v", string(body))

	decodeerr := json.Unmarshal(body, &permission)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("Project Group Permission decoded is: %#v", permission)

	d.Set("permission", permission.Permission)
	d.Set("group_slug", permission.Group.Slug)
	d.Set("workspace", permission.Group.Workspace.Slug)
	d.Set("project_key", projectKey)

	return nil
}

func resourceProjectGroupPermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, projectKey, groupSlug, err := projectGroupPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Delete(fmt.Sprintf("2.0/workspaces/%s/projects/%s/permissions-config/groups/%s",
		workspace,
		projectKey,
		groupSlug,
	))

	return diag.FromErr(err)
}

func projectGroupPermissionId(id string) (string, string, string, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE:PROJECT-KEY:GROUP-SLUG", id)
	}

	return parts[0], parts[1], parts[2], nil
}
