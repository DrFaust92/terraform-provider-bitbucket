package bitbucket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceWorkspaceVariable() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceWorkspaceVariableCreate,
		UpdateWithoutTimeout: resourceWorkspaceVariableUpdate,
		ReadWithoutTimeout:   resourceWorkspaceVariableRead,
		DeleteWithoutTimeout: resourceWorkspaceVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"secured": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func newWorkspaceVariableFromResource(d *schema.ResourceData) bitbucket.PipelineVariable {
	dk := bitbucket.PipelineVariable{
		Key:     d.Get("key").(string),
		Value:   d.Get("value").(string),
		Secured: d.Get("secured").(bool),
	}
	return dk
}

func resourceWorkspaceVariableCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi
	rvcr := newWorkspaceVariableFromResource(d)

	workspacePipeBody := &bitbucket.PipelinesApiCreatePipelineVariableForWorkspaceOpts{
		Body: optional.NewInterface(rvcr),
	}

	workspace := d.Get("workspace").(string)

	log.Printf("[DEBUG] Workspace Variable Request: %#v", workspacePipeBody)

	rvRes, res, err := pipeApi.CreatePipelineVariableForWorkspace(c.AuthContext, workspace, workspacePipeBody)

	log.Printf("[DEBUG] Workspace Variable Create Request Res: %#v", res)

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", workspace, rvRes.Uuid))

	return resourceWorkspaceVariableRead(ctx, d, m)
}

func resourceWorkspaceVariableRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, uuid, err := workspaceVarId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	rvRes, res, err := pipeApi.GetPipelineVariableForWorkspace(c.AuthContext, workspace, uuid)

	log.Printf("[DEBUG] Workspace Variable Get Request Res: %#v", res)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Workspace Variable (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("uuid", rvRes.Uuid)
	d.Set("workspace", workspace)
	d.Set("key", rvRes.Key)
	d.Set("secured", rvRes.Secured)

	if !rvRes.Secured {
		d.Set("value", rvRes.Value)
	} else {
		d.Set("value", d.Get("value").(string))
	}

	return nil
}

func resourceWorkspaceVariableUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, uuid, err := workspaceVarId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	rvcr := newWorkspaceVariableFromResource(d)

	_, res, err := pipeApi.UpdatePipelineVariableForWorkspace(c.AuthContext, rvcr, workspace, uuid)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return resourceWorkspaceVariableRead(ctx, d, m)
}

func resourceWorkspaceVariableDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, uuid, err := workspaceVarId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := pipeApi.DeletePipelineVariableForWorkspace(c.AuthContext, workspace, uuid)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func workspaceVarId(workspace string) (string, string, error) {
	idparts := strings.Split(workspace, "/")
	if len(idparts) == 2 {
		return idparts[0], idparts[1], nil
	} else {
		return "", "", fmt.Errorf("incorrect ID format, should match `workspace/uuid`")
	}
}
