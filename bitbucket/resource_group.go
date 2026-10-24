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

type UserGroup struct {
	Name                    string `json:"name,omitempty"`
	Slug                    string `json:"slug,omitempty"`
	AutoAdd                 bool   `json:"auto_add,omitempty"`
	Permission              string `json:"permission,omitempty"`
	EmailForwardingDisabled bool   `json:"email_forwarding_disabled,omitempty"`
}

func resourceGroup() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceGroupsCreate,
		ReadWithoutTimeout:   resourceGroupsRead,
		UpdateWithoutTimeout: resourceGroupsUpdate,
		DeleteWithoutTimeout: resourceGroupsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auto_add": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"permission": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"read", "write", "admin"}, false),
			},
			"email_forwarding_disabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceGroupsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	group := expandGroup(d)
	log.Printf("[DEBUG] Group Request: %#v", group)

	workspace := d.Get("workspace").(string)
	body := []byte(fmt.Sprintf("name=%s", group.Name))
	groupReq, err := client.PostNonJson(fmt.Sprintf("1.0/groups/%s", workspace), bytes.NewBuffer(body))
	if err != nil {
		return diag.FromErr(err)
	}

	body, readerr := io.ReadAll(groupReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Group Req Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &group)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Group Req Response Decoded: %#v", group)

	d.SetId(fmt.Sprintf("%s/%s", workspace, group.Slug))

	return resourceGroupsRead(ctx, d, m)
}

func resourceGroupsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, slug, err := groupId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	groupsReq, err := client.Get(fmt.Sprintf("1.0/groups/%s/%s", workspace, slug))

	if groupsReq != nil && groupsReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	if groupsReq.Body == nil {
		return diag.Errorf("error reading Group (%s): empty response", d.Id())
	}

	var grp *UserGroup

	body, readerr := io.ReadAll(groupsReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Groups Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &grp)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Groups Response Decoded: %#v", grp)

	d.Set("workspace", workspace)
	d.Set("slug", grp.Slug)
	d.Set("name", grp.Name)
	d.Set("auto_add", grp.AutoAdd)
	d.Set("permission", grp.Permission)
	d.Set("email_forwarding_disabled", grp.EmailForwardingDisabled)

	return nil
}

func resourceGroupsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	group := expandGroup(d)
	log.Printf("[DEBUG] Group Request: %#v", group)
	bytedata, err := json.Marshal(group)

	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Put(fmt.Sprintf("1.0/groups/%s/%s/",
		d.Get("workspace").(string), d.Get("slug").(string)), bytes.NewBuffer(bytedata))

	if err != nil {
		return diag.FromErr(err)
	}

	return resourceGroupsRead(ctx, d, m)
}

func resourceGroupsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, slug, err := groupId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Delete(fmt.Sprintf("1.0/groups/%s/%s", workspace, slug))

	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}

func expandGroup(d *schema.ResourceData) *UserGroup {
	group := &UserGroup{
		Name: d.Get("name").(string),
	}

	if v, ok := d.GetOk("auto_add"); ok {
		group.AutoAdd = v.(bool)
	}

	if v, ok := d.GetOk("permission"); ok && v.(string) != "" {
		group.Permission = v.(string)
	}

	if v, ok := d.GetOk("email_forwarding_disabled"); ok {
		group.EmailForwardingDisabled = v.(bool)
	}

	return group
}

func groupId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE-ID/GROUP-SLUG-ID", id)
	}

	return parts[0], parts[1], nil
}
