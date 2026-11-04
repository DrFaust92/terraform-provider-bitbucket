package bitbucket

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataDeployments() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadDeployments,

		Schema: map[string]*schema.Schema{
			"uuids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
			},
			"names": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataReadDeployments(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	deployApi := c.ApiClient.DeploymentsApi

	workspace := d.Get("workspace").(string)
	repoId := d.Get("repository").(string)

	client := m.(Clients).httpClient
	var uuids, names []string
	next := ""
	for {
		var page bitbucket.PaginatedEnvironments
		var res *http.Response
		var err error
		if next == "" {
			page, res, err = deployApi.GetEnvironmentsForRepository(c.AuthContext, workspace, repoId, nil)
		} else {
			res, err = client.GetNext(next, &page)
		}
		if err := handleClientError(res, err); err != nil {
			return diag.FromErr(err)
		}

		for _, deployment := range page.Values {
			uuids = append(uuids, deployment.Uuid)
			names = append(names, deployment.Name)
		}

		if page.Next == "" {
			break
		}
		next = page.Next
	}

	d.SetId(fmt.Sprintf("%s/%s", workspace, repoId))
	d.Set("uuids", uuids)
	d.Set("names", names)

	return nil
}
