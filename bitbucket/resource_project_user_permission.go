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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

type ProjectUserPermission struct {
	Permission string       `json:"permission"`
	User       *ProjectUser `json:"user,omitempty"`
}

type ProjectUser struct {
	UUID string `json:"uuid,omitempty"`
}

func resourceProjectUserPermission() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceProjectUserPermissionPut,
		ReadWithoutTimeout:   resourceProjectUserPermissionRead,
		UpdateWithoutTimeout: resourceProjectUserPermissionPut,
		DeleteWithoutTimeout: resourceProjectUserPermissionDelete,
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
			"user_id": {
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

func createProjectUserPermission(d *schema.ResourceData) *ProjectUserPermission {

	permission := &ProjectUserPermission{
		Permission: d.Get("permission").(string),
	}

	return permission
}

func resourceProjectUserPermissionPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	permission := createProjectUserPermission(d)

	payload, err := json.Marshal(permission)
	if err != nil {
		return diag.FromErr(err)
	}

	workspace := d.Get("workspace").(string)
	projectKey := d.Get("project_key").(string)
	userSlug := d.Get("user_id").(string)

	permissionReq, err := client.Put(fmt.Sprintf("2.0/workspaces/%s/projects/%s/permissions-config/users/%s",
		workspace,
		projectKey,
		userSlug,
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
		d.SetId(fmt.Sprintf("%s:%s:%s", workspace, projectKey, userSlug))
	}

	return resourceProjectUserPermissionRead(ctx, d, m)
}

func resourceProjectUserPermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, projectKey, userSlug, err := ProjectUserPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	permissionReq, err := client.Get(fmt.Sprintf("2.0/workspaces/%s/projects/%s/permissions-config/users/%s",
		workspace,
		projectKey,
		userSlug,
	))

	if permissionReq != nil && permissionReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Project User Permission (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	var permission ProjectUserPermission

	body, readerr := io.ReadAll(permissionReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("Project User Permission raw is: %#v", string(body))

	decodeerr := json.Unmarshal(body, &permission)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("Project User Permission decoded is: %#v", permission)

	d.Set("permission", permission.Permission)
	d.Set("user_id", permission.User.UUID)
	d.Set("workspace", workspace)
	d.Set("project_key", projectKey)

	return nil
}

func resourceProjectUserPermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, projectKey, userSlug, err := ProjectUserPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Delete(fmt.Sprintf("2.0/workspaces/%s/projects/%s/permissions-config/users/%s",
		workspace,
		projectKey,
		userSlug,
	))

	return diag.FromErr(err)
}

func ProjectUserPermissionId(id string) (string, string, string, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE:REPO-SLUG:GROUP-SLUG", id)
	}

	return parts[0], parts[1], parts[2], nil
}
