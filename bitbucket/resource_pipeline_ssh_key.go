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

func resourcePipelineSshKey() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourcePipelineSshKeysPut,
		ReadWithoutTimeout:   resourcePipelineSshKeysRead,
		UpdateWithoutTimeout: resourcePipelineSshKeysPut,
		DeleteWithoutTimeout: resourcePipelineSshKeysDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"private_key": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"public_key": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourcePipelineSshKeysPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	pipeSshKey := expandPipelineSshKey(d)
	log.Printf("[DEBUG] Pipeline Ssh Key Request: %#v", pipeSshKey)

	repo := d.Get("repository").(string)
	workspace := d.Get("workspace").(string)
	_, res, err := pipeApi.UpdateRepositoryPipelineKeyPair(c.AuthContext, *pipeSshKey, workspace, repo)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", workspace, repo))

	return resourcePipelineSshKeysRead(ctx, d, m)
}

func resourcePipelineSshKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, err := pipeSshKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	key, res, err := pipeApi.GetRepositoryPipelineSshKeyPair(c.AuthContext, workspace, repo)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Pipeline Ssh Key (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("repository", repo)
	d.Set("workspace", workspace)
	d.Set("public_key", key.PublicKey)
	d.Set("private_key", d.Get("private_key").(string))

	return nil
}

func resourcePipelineSshKeysDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, err := pipeSshKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := pipeApi.DeleteRepositoryPipelineKeyPair(c.AuthContext, workspace, repo)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}

func expandPipelineSshKey(d *schema.ResourceData) *bitbucket.PipelineSshKeyPair {
	key := &bitbucket.PipelineSshKeyPair{
		PublicKey:  d.Get("public_key").(string),
		PrivateKey: d.Get("private_key").(string),
	}

	return key
}

func pipeSshKeyId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE-ID/REPO-ID", id)
	}

	return parts[0], parts[1], nil
}
