package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type UserGroupMembership struct {
	UUID string `json:"uuid,omitempty"`
}

func resourceGroupMembership() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceGroupMembershipsPut,
		ReadWithoutTimeout:   resourceGroupMembershipsRead,
		DeleteWithoutTimeout: resourceGroupMembershipsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"group_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"uuid": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceGroupMembershipsPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace := d.Get("workspace").(string)
	groupSlug := d.Get("group_slug").(string)
	uuid := d.Get("uuid").(string)

	_, err := client.PutOnly(fmt.Sprintf("1.0/groups/%s/%s/members/%s",
		workspace, groupSlug, uuid))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", workspace, groupSlug, uuid))

	return resourceGroupMembershipsRead(ctx, d, m)
}

func resourceGroupMembershipsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, slug, uuid, err := groupMemberId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	groupsReq, _ := client.Get(fmt.Sprintf("1.0/groups/%s/%s/members", workspace, slug))

	if groupsReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Group Membership (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if groupsReq.Body == nil {
		return diag.Errorf("error reading Group Membership (%s): empty response", d.Id())
	}

	var members []*UserGroupMembership

	body, readerr := io.ReadAll(groupsReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Group Membership Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &members)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Group Membership Response Decoded: %#v", members)

	var member *UserGroupMembership
	for _, mbr := range members {
		if mbr.UUID == uuid {
			member = mbr
			break
		}
	}

	if member == nil {
		if d.IsNewResource() {
			return diag.Errorf("error getting Group Member (%s): not found", d.Id())
		}

		log.Printf("[WARN] Group Member (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Group Member Response Decoded: %#v", member)

	d.Set("workspace", workspace)
	d.Set("group_slug", slug)
	d.Set("uuid", member.UUID)

	return nil
}

func resourceGroupMembershipsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, slug, uuid, err := groupMemberId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Delete(fmt.Sprintf("1.0/groups/%s/%s/members/%s",
		workspace, slug, uuid))

	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}

func groupMemberId(id string) (string, string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE-ID/GROUP-SLUG-ID/MEMBER-UUID", id)
	}

	return parts[0], parts[1], parts[2], nil
}
