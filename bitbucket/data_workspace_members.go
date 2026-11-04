package bitbucket

import (
	"context"
	"log"
	"net/http"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataWorkspaceMembers() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadWorkspaceMembers,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"members": {
				Type:       schema.TypeSet,
				Elem:       &schema.Schema{Type: schema.TypeString},
				Computed:   true,
				Deprecated: "use workspace_members instead",
			},
			"workspace_members": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"username": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"display_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataReadWorkspaceMembers(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient

	workspaceApi := c.ApiClient.WorkspacesApi

	workspace := d.Get("workspace").(string)

	var members []string
	var accounts []bitbucket.Account
	options := bitbucket.WorkspacesApiWorkspacesWorkspaceMembersGetOpts{}

	client := m.(Clients).httpClient
	next := ""
	for {
		var flattenAccountsReq bitbucket.PaginatedWorkspaceMemberships
		var res *http.Response
		var err error
		if next == "" {
			flattenAccountsReq, res, err = workspaceApi.WorkspacesWorkspaceMembersGet(c.AuthContext, workspace, &options)
		} else {
			res, err = client.GetNext(next, &flattenAccountsReq)
		}
		if err := handleClientError(res, err); err != nil {
			return diag.FromErr(err)
		}

		for _, member := range flattenAccountsReq.Values {
			members = append(members, member.User.Uuid)
			accounts = append(accounts, *member.User)
		}

		if flattenAccountsReq.Next == "" {
			break
		}
		next = flattenAccountsReq.Next
	}

	d.SetId(workspace)
	d.Set("workspace", workspace)
	d.Set("members", members)
	d.Set("workspace_members", flattenAccounts(accounts))

	return nil
}

func flattenAccounts(flattenAccounts []bitbucket.Account) []interface{} {
	if len(flattenAccounts) == 0 {
		return nil
	}

	var tfList []interface{}

	for _, account := range flattenAccounts {
		log.Printf("[DEBUG] Workspace Member Response: %#v", account)

		flattenAccounts := map[string]interface{}{
			"uuid":         account.Uuid,
			"username":     account.DisplayName,
			"display_name": account.Username,
		}

		tfList = append(tfList, flattenAccounts)
	}

	return tfList
}
