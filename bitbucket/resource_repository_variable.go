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

func resourceRepositoryVariable() *schema.Resource {
	valueWO, valueWOVersion := writeOnlySchemas("value")

	return &schema.Resource{
		CreateWithoutTimeout: resourceRepositoryVariableCreate,
		UpdateWithoutTimeout: resourceRepositoryVariableUpdate,
		ReadWithoutTimeout:   resourceRepositoryVariableRead,
		DeleteWithoutTimeout: resourceRepositoryVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceRepositoryVariableImport,
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
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ExactlyOneOf: []string{"value", "value_wo"},
			},
			"value_wo":         valueWO,
			"value_wo_version": valueWOVersion,
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
		Value:   writeOnlyOrValue(d, "value"),
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

	// With value_wo the value must not be stored in state.
	if !rvRes.Secured && !usesWriteOnly(d, "value") {
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

// resourceRepositoryVariableImport accepts WORKSPACE/REPO-SLUG/KEY/UUID, or WORKSPACE/REPO-SLUG/KEY and looks the
// UUID up by key, which is unique within a repository.
func resourceRepositoryVariableImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	idParts := strings.Split(d.Id(), "/")
	for _, part := range idParts {
		if part == "" {
			idParts = nil
			break
		}
	}
	if len(idParts) != 3 && len(idParts) != 4 {
		return nil, fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE/REPO-SLUG/KEY or WORKSPACE/REPO-SLUG/KEY/UUID", d.Id())
	}

	workspace, repoSlug, key := idParts[0], idParts[1], idParts[2]
	var uuid string
	if len(idParts) == 4 {
		uuid = idParts[3]
	} else {
		vars, err := listRepositoryVariables(m.(Clients).genClient, workspace, repoSlug)
		if err != nil {
			return nil, err
		}
		for _, v := range vars {
			if v.Key == key {
				uuid = v.Uuid
				break
			}
		}
		if uuid == "" {
			return nil, fmt.Errorf("repository variable %q not found in %s/%s", key, workspace, repoSlug)
		}
	}

	d.SetId(key)
	d.Set("uuid", uuid)
	d.Set("repository", fmt.Sprintf("%s/%s", workspace, repoSlug))
	return []*schema.ResourceData{d}, nil
}

// listRepositoryVariables returns every pipeline variable of a repository, following pages.
func listRepositoryVariables(c ProviderConfig, workspace, repoSlug string) ([]bitbucket.PipelineVariable, error) {
	pipeApi := c.ApiClient.PipelinesApi

	var all []bitbucket.PipelineVariable
	for page := int32(1); ; page++ {
		opts := &bitbucket.PipelinesApiGetRepositoryPipelineVariablesOpts{Page: optional.NewInt32(page)}
		vars, res, err := pipeApi.GetRepositoryPipelineVariables(c.AuthContext, workspace, repoSlug, opts)
		if err := handleClientError(res, err); err != nil {
			return nil, err
		}

		all = append(all, vars.Values...)

		if vars.Next == "" || len(vars.Values) == 0 {
			return all, nil
		}
	}
}

func repoVarId(repo string) (string, string, error) {
	idparts := strings.Split(repo, "/")
	if len(idparts) == 2 {
		return idparts[0], idparts[1], nil
	} else {
		return "", "", fmt.Errorf("incorrect ID format, should match `owner/key`")
	}
}
