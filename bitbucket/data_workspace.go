package bitbucket

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataWorkspace() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataReadWorkspace,

		Schema: map[string]*schema.Schema{
			"is_personal": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the workspace belongs to an individual user.",
			},
			"is_privacy_enforced": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the workspace only allows private content.",
			},
			"forking_mode": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The rules for forking repositories in the workspace: `allow_forks`, `internal_only` or `no_public_forks`.",
			},
			"enforced_signed_commits": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether signed commits are required for the workspace.",
			},
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"is_private": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func dataReadWorkspace(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient

	workspaceApi := c.ApiClient.WorkspacesApi

	workspace := d.Get("workspace").(string)
	workspaceReq, res, err := workspaceApi.WorkspacesWorkspaceGet(c.AuthContext, workspace)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(workspaceReq.Uuid)
	d.Set("workspace", workspace)
	d.Set("name", workspaceReq.Name)
	d.Set("slug", workspaceReq.Slug)
	d.Set("is_private", workspaceReq.IsPrivate)
	d.Set("is_personal", workspaceReq.IsPersonal)
	d.Set("is_privacy_enforced", workspaceReq.IsPrivacyEnforced)
	d.Set("forking_mode", workspaceReq.ForkingMode)
	d.Set("enforced_signed_commits", workspaceReq.EnforcedSignedCommits)

	return nil
}
