package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataDeployment() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadDeployment,

		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Required: true,
			},
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"stage": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataReadDeployment(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).httpClient

	workspace := d.Get("workspace").(string)
	repoId := d.Get("repository").(string)

	res, err := c.Get(fmt.Sprintf("2.0/repositories/%s/%s/environments/%s",
		workspace,
		repoId,
		d.Get("uuid").(string),
	))
	if err != nil {
		return diag.FromErr(err)
	}

	if res.StatusCode == http.StatusNotFound {
		return diag.Errorf("deployment not found")
	}

	if res.StatusCode >= http.StatusInternalServerError {
		return diag.Errorf("internal server error fetching deployment")
	}

	var deploy Deployment
	body, readerr := io.ReadAll(res.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Deployment response raw: %s", string(body))

	decodeerr := json.Unmarshal(body, &deploy)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Deployment response: %#v", deploy)

	d.SetId(deploy.UUID)
	d.Set("uuid", deploy.UUID)
	d.Set("name", deploy.Name)
	d.Set("stage", deploy.Stage.Name)
	d.Set("repository", repoId)
	d.Set("workspace", workspace)

	return nil
}
