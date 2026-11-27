package bitbucket

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataRepositoryVariables() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadRepositoryVariables,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
			},
			"variables": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"value": {
							Type:      schema.TypeString,
							Computed:  true,
							Sensitive: true,
						},
						"secured": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataReadRepositoryVariables(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	workspace := d.Get("workspace").(string)
	repoSlug := d.Get("repository").(string)

	vars, err := listRepositoryVariables(m.(Clients).genClient, workspace, repoSlug)
	if err != nil {
		return diag.FromErr(err)
	}

	sort.Slice(vars, func(i, j int) bool { return vars[i].Key < vars[j].Key })

	variables := make([]interface{}, 0, len(vars))
	for _, v := range vars {
		// The API never returns the value of a secured variable.
		variables = append(variables, map[string]interface{}{
			"uuid":    v.Uuid,
			"key":     v.Key,
			"value":   v.Value,
			"secured": v.Secured,
		})
	}

	d.SetId(fmt.Sprintf("%s/%s", workspace, repoSlug))
	if err := d.Set("variables", variables); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
