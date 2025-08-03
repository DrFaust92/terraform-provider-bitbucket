package bitbucket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRepositoryVariable() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceRepositoryVariableCreate,
		UpdateWithoutTimeout: resourceRepositoryVariableUpdate,
		ReadWithoutTimeout:   resourceRepositoryVariableRead,
		DeleteWithoutTimeout: resourceRepositoryVariableDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
					return nil, fmt.Errorf("unexpected format of ID (%q), expected REPOSITORY/KEY/UUID", d.Id())
				}
				d.SetId(idParts[2])
				d.Set("uuid", idParts[3])
				d.Set("repository", fmt.Sprintf("%s/%s", idParts[0], idParts[1]))
				return []*schema.ResourceData{d}, nil
			},
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
			"repository": {
				Type:     schema.TypeString,
				Required: true,
			},
			"workspace": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func newRepositoryVariableFromResource(d *schema.ResourceData) bitbucket.PipelineVariable {
	dk := bitbucket.PipelineVariable{
		Key:     d.Get("key").(string),
		Value:   d.Get("value").(string),
		Secured: d.Get("secured").(bool),
	}
	return dk
}

func resourceRepositoryVariableCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi
	rvcr := newRepositoryVariableFromResource(d)

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return diag.FromErr(err)
	}

	rvRes, res, err := pipeApi.CreateRepositoryPipelineVariable(c.AuthContext, rvcr, workspace, repoSlug)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("uuid", rvRes.Uuid)
	d.SetId(rvRes.Key)

	return resourceRepositoryVariableRead(ctx, d, m)
}

func resourceRepositoryVariableRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return diag.FromErr(err)
	}

	rvRes, res, err := pipeApi.GetRepositoryPipelineVariable(c.AuthContext, workspace, repoSlug, d.Get("uuid").(string))

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository Variable (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("uuid", rvRes.Uuid)
	d.Set("key", rvRes.Key)
	d.Set("secured", rvRes.Secured)
	d.Set("workspace", workspace)

	if !rvRes.Secured {
		d.Set("value", rvRes.Value)
	} else {
		d.Set("value", d.Get("value").(string))
	}

	return nil
}

func resourceRepositoryVariableUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return diag.FromErr(err)
	}

	rvcr := newRepositoryVariableFromResource(d)

	_, res, err := pipeApi.UpdateRepositoryPipelineVariable(c.AuthContext, rvcr, workspace, repoSlug, d.Get("uuid").(string))
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return resourceRepositoryVariableRead(ctx, d, m)
}

func resourceRepositoryVariableDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := pipeApi.DeleteRepositoryPipelineVariable(c.AuthContext, workspace, repoSlug, d.Get("uuid").(string))
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func repoVarId(repo string) (string, string, error) {
	idparts := strings.Split(repo, "/")
	if len(idparts) == 2 {
		return idparts[0], idparts[1], nil
	} else {
		return "", "", fmt.Errorf("incorrect ID format, should match `owner/key`")
	}
}
