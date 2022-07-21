package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/migara/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourcePipelineSshKey() *schema.Resource {
	return &schema.Resource{
		Create: resourcePipelineSshKeysPut,
		Read:   resourcePipelineSshKeysRead,
		Update: resourcePipelineSshKeysPut,
		Delete: resourcePipelineSshKeysDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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

func resourcePipelineSshKeysPut(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	pipeSshKey := expandPipelineSshKey(d)
	log.Printf("[DEBUG] Pipeline Ssh Key Request: %#v", pipeSshKey)

	repo := d.Get("repository").(string)
	workspace := d.Get("workspace").(string)
	_, _, err := pipeApi.UpdateRepositoryPipelineKeyPair(c.AuthContext, *pipeSshKey, workspace, repo)

	if err != nil {
		return fmt.Errorf("error creating pipeline ssh key: %w", err)
	}

	d.SetId(string(fmt.Sprintf("%s/%s", workspace, repo)))

	return resourcePipelineSshKeysRead(d, m)
}

func resourcePipelineSshKeysRead(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, err := pipeSshKeyId(d.Id())
	if err != nil {
		return err
	}

	key, res, err := pipeApi.GetRepositoryPipelineSshKeyPair(c.AuthContext, workspace, repo)
	if err != nil {
		return fmt.Errorf("error reading Pipeline Ssh Key (%s): %w", d.Id(), err)
	}

	if res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Pipeline Ssh Key (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if res.Body == nil {
		return fmt.Errorf("error getting Pipeline Ssh Key (%s): empty response", d.Id())
	}

	d.Set("repository", repo)
	d.Set("workspace", workspace)
	d.Set("public_key", key.PublicKey)
	d.Set("private_key", d.Get("private_key").(string))

	return nil
}

func resourcePipelineSshKeysDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, err := pipeSshKeyId(d.Id())
	if err != nil {
		return err
	}

	_, err = pipeApi.DeleteRepositoryPipelineKeyPair(c.AuthContext, workspace, repo)

	if err != nil {
		return fmt.Errorf("error deleting Pipeline Ssh Key (%s): %w", d.Id(), err)
	}

	return err
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
