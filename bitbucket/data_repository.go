package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataRepository() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadRepository,

		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
			},
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func dataReadRepository(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).httpClient

	slug := d.Get("slug").(string)
	workspace := d.Get("workspace").(string)

	res, err := c.Get(fmt.Sprintf("2.0/repositories/%s/%s",
		workspace,
		slug))

	if err != nil {
		return diag.FromErr(err)
	}

	if res.StatusCode == http.StatusNotFound {
		return diag.Errorf(("repository not found"))
	}

	if res.StatusCode == http.StatusInternalServerError {
		return diag.Errorf("internal server error fetching repository")
	}

	var repository bitbucket.Repository
	body, readerr := io.ReadAll(res.Body)

	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Deployment response raw: %s", string(body))

	decodeerr := json.Unmarshal(body, &repository)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Deployment response: %#v", repository)

	d.SetId(fmt.Sprintf("%s/%s", workspace, slug))
	d.Set("workspace", workspace)
	d.Set("slug", slug)

	return nil
}
