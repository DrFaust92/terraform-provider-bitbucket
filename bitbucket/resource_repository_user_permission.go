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

type RepositoryUserPermission struct {
	Permission string          `json:"permission"`
	User       *RepositoryUser `json:"user,omitempty"`
}

type RepositoryUser struct {
	UUID string `json:"uuid,omitempty"`
}

func resourceRepositoryUserPermission() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceRepositoryUserPermissionPut,
		ReadWithoutTimeout:   resourceRepositoryUserPermissionRead,
		UpdateWithoutTimeout: resourceRepositoryUserPermissionPut,
		DeleteWithoutTimeout: resourceRepositoryUserPermissionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repo_slug": {
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
				ValidateFunc: validation.StringInSlice([]string{"admin", "write", "read", "none"}, false),
			},
		},
	}
}

func createRepositoryUserPermission(d *schema.ResourceData) *RepositoryUserPermission {

	permission := &RepositoryUserPermission{
		Permission: d.Get("permission").(string),
	}

	return permission
}

func resourceRepositoryUserPermissionPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	permission := createRepositoryUserPermission(d)

	payload, err := json.Marshal(permission)
	if err != nil {
		return diag.FromErr(err)
	}

	workspace := d.Get("workspace").(string)
	repoSlug := d.Get("repo_slug").(string)
	userSlug := d.Get("user_id").(string)

	permissionReq, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s/permissions-config/users/%s",
		workspace,
		repoSlug,
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
		d.SetId(fmt.Sprintf("%s:%s:%s", workspace, repoSlug, userSlug))
	}

	if userSlug != permission.User.UUID {
		return diag.FromErr(fmt.Errorf("the user_id must be a UUID, but a user name was given (\"%s\"). The UUID for this user is \"%s\"", userSlug, permission.User.UUID))
	}

	return resourceRepositoryUserPermissionRead(ctx, d, m)
}

func resourceRepositoryUserPermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, repoSlug, userSlug, err := repositoryUserPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	permissionReq, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/permissions-config/users/%s",
		workspace,
		repoSlug,
		userSlug,
	))

	if permissionReq != nil && permissionReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository User Permission (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	var permission RepositoryUserPermission

	body, readerr := io.ReadAll(permissionReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("Repository User Permission raw is: %#v", string(body))

	decodeerr := json.Unmarshal(body, &permission)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("Repository User Permission decoded is: %#v", permission)

	d.Set("permission", permission.Permission)
	d.Set("user_id", permission.User.UUID)
	d.Set("workspace", workspace)
	d.Set("repo_slug", repoSlug)

	return nil
}

func resourceRepositoryUserPermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, repoSlug, userSlug, err := repositoryUserPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Delete(fmt.Sprintf("2.0/repositories/%s/%s/permissions-config/users/%s",
		workspace,
		repoSlug,
		userSlug,
	))

	return diag.FromErr(err)
}

func repositoryUserPermissionId(id string) (string, string, string, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE:REPO-SLUG:GROUP-SLUG", id)
	}

	return parts[0], parts[1], parts[2], nil
}
