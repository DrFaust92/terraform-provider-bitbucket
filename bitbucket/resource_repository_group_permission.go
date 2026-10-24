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

type RepositoryGroupPermission struct {
	Permission string           `json:"permission"`
	Group      *RepositoryGroup `json:"group,omitempty"`
}

type RepositoryGroup struct {
	Name      string              `json:"name,omitempty"`
	Slug      string              `json:"slug,omitempty"`
	Workspace bitbucket.Workspace `json:"workspace,omitempty"`
}

func resourceRepositoryGroupPermission() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceRepositoryGroupPermissionPut,
		ReadWithoutTimeout:   resourceRepositoryGroupPermissionRead,
		UpdateWithoutTimeout: resourceRepositoryGroupPermissionPut,
		DeleteWithoutTimeout: resourceRepositoryGroupPermissionDelete,
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
			"group_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"permission": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"admin", "write", "read"}, false),
			},
		},
	}
}

func createRepositoryGroupPermission(d *schema.ResourceData) *RepositoryGroupPermission {

	permission := &RepositoryGroupPermission{
		Permission: d.Get("permission").(string),
	}

	return permission
}

func resourceRepositoryGroupPermissionPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	permission := createRepositoryGroupPermission(d)

	payload, err := json.Marshal(permission)
	if err != nil {
		return diag.FromErr(err)
	}

	workspace := d.Get("workspace").(string)
	repoSlug := d.Get("repo_slug").(string)
	groupSlug := d.Get("group_slug").(string)

	permissionReq, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s/permissions-config/groups/%s",
		workspace,
		repoSlug,
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
		d.SetId(fmt.Sprintf("%s:%s:%s", workspace, repoSlug, groupSlug))
	}

	return resourceRepositoryGroupPermissionRead(ctx, d, m)
}

func resourceRepositoryGroupPermissionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, repoSlug, groupSlug, err := repositoryGroupPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	permissionReq, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/permissions-config/groups/%s",
		workspace,
		repoSlug,
		groupSlug,
	))

	if permissionReq != nil && permissionReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository Group Permission (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	var permission RepositoryGroupPermission

	body, readerr := io.ReadAll(permissionReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("Repository Group Permission raw is: %#v", string(body))

	decodeerr := json.Unmarshal(body, &permission)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("Repository Group Permission decoded is: %#v", permission)

	d.Set("permission", permission.Permission)
	d.Set("group_slug", permission.Group.Slug)
	d.Set("workspace", permission.Group.Workspace.Slug)
	d.Set("repo_slug", repoSlug)

	return nil
}

func resourceRepositoryGroupPermissionDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, repoSlug, groupSlug, err := repositoryGroupPermissionId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Delete(fmt.Sprintf("2.0/repositories/%s/%s/permissions-config/groups/%s",
		workspace,
		repoSlug,
		groupSlug,
	))

	return diag.FromErr(err)
}

func repositoryGroupPermissionId(id string) (string, string, string, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE:REPO-SLUG:GROUP-SLUG", id)
	}

	return parts[0], parts[1], parts[2], nil
}
