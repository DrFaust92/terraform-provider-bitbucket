package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataGroup() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadGroup,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
			},
			"auto_add": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"permission": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"email_forwarding_disabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataReadGroup(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace := d.Get("workspace").(string)
	slug := d.Get("slug").(string)

	groupsReq, err := client.Get(fmt.Sprintf("1.0/groups/%s/%s", workspace, slug))

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

	log.Printf("[DEBUG] Group Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &grp)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Group Response Decoded: %#v", grp)

	d.SetId(fmt.Sprintf("%s/%s", workspace, slug))
	d.Set("workspace", workspace)
	d.Set("slug", grp.Slug)
	d.Set("name", grp.Name)
	d.Set("auto_add", grp.AutoAdd)
	d.Set("permission", grp.Permission)
	d.Set("email_forwarding_disabled", grp.EmailForwardingDisabled)

	return nil
}
