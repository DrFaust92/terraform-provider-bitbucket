package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceDeployKey() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceDeployKeysCreate,
		ReadWithoutTimeout:   resourceDeployKeysRead,
		UpdateWithoutTimeout: resourceDeployKeysUpdate,
		DeleteWithoutTimeout: resourceDeployKeysDelete,
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
			"key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"label": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"comment": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDeployKeysCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	deployKey := expandsshKey(d)
	log.Printf("[DEBUG] Deploy Key Request: %#v", deployKey)
	bytedata, err := json.Marshal(deployKey)

	if err != nil {
		return diag.FromErr(err)
	}

	repo := d.Get("repository").(string)
	workspace := d.Get("workspace").(string)
	deployKeyReq, err := client.Post(fmt.Sprintf("2.0/repositories/%s/%s/deploy-keys", workspace, repo), bytes.NewBuffer(bytedata))

	if err != nil {
		return diag.FromErr(err)
	}

	body, readerr := io.ReadAll(deployKeyReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Deploy Keys Create Response JSON: %v", string(body))

	var deployKeyRes SshKey

	decodeerr := json.Unmarshal(body, &deployKeyRes)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] Deploy Keys Create Response Decoded: %#v", deployKeyRes)

	d.SetId(fmt.Sprintf("%s/%s/%d", workspace, repo, deployKeyRes.ID))

	return resourceDeployKeysRead(ctx, d, m)
}

func resourceDeployKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	deployApi := c.ApiClient.DeploymentsApi

	workspace, repo, keyId, err := deployKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	deployKey, res, err := deployApi.RepositoriesWorkspaceRepoSlugDeployKeysKeyIdGet(c.AuthContext, keyId, repo, workspace)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Deploy Key (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deploy Key Response: %#v", deployKey)

	d.Set("repository", repo)
	d.Set("workspace", workspace)
	d.Set("key", d.Get("key").(string))
	d.Set("label", deployKey.Label)
	d.Set("comment", deployKey.Comment)
	d.Set("key_id", keyId)

	return nil
}

func resourceDeployKeysUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	deployKey := expandsshKey(d)
	log.Printf("[DEBUG] Deploy Key Request: %#v", deployKey)
	bytedata, err := json.Marshal(deployKey)

	if err != nil {
		return diag.FromErr(err)
	}

	workspace, repo, keyId, err := deployKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/deploy-keys/%s",
		workspace, repo, keyId), bytes.NewBuffer(bytedata))

	if err != nil {
		return diag.Errorf("error updating Deploy Key (%s): %s", d.Id(), err)
	}

	return resourceDeployKeysRead(ctx, d, m)
}

func resourceDeployKeysDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	deployApi := c.ApiClient.DeploymentsApi

	workspace, repo, keyId, err := deployKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := deployApi.RepositoriesWorkspaceRepoSlugDeployKeysKeyIdDelete(c.AuthContext, keyId, repo, workspace)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}

func deployKeyId(id string) (string, string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE-ID/REPO-ID/KEY-ID", id)
	}

	return parts[0], parts[1], parts[2], nil
}
