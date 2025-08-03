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

// sshKey is the data we need to send to create a new SSH Key for the repository
type SshKey struct {
	ID      int    `json:"id,omitempty"`
	UUID    string `json:"uuid,omitempty"`
	Key     string `json:"key,omitempty"`
	Label   string `json:"label,omitempty"`
	Comment string `json:"comment,omitempty"`
}

func resourceSshKey() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceSshKeysCreate,
		ReadWithoutTimeout:   resourceSshKeysRead,
		UpdateWithoutTimeout: resourceSshKeysUpdate,
		DeleteWithoutTimeout: resourceSshKeysDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"user": {
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
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"comment": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSshKeysCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	sshKey := expandsshKey(d)

	sshKeyBody := &bitbucket.SshApiUsersSelectedUserSshKeysPostOpts{
		Body: optional.NewInterface(sshKey),
	}

	user := d.Get("user").(string)
	sshKeyReq, res, err := sshApi.UsersSelectedUserSshKeysPost(c.AuthContext, user, sshKeyBody)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", user, sshKeyReq.Uuid))

	return resourceSshKeysRead(ctx, d, m)
}

func resourceSshKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	user, keyId, err := sshKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	sshKeyReq, res, err := sshApi.UsersSelectedUserSshKeysKeyIdGet(c.AuthContext, keyId, user)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] SSH Key (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	if res.Body == nil {
		return diag.Errorf("error getting SSH Key (%s): empty response", d.Id())
	}

	d.Set("user", user)
	d.Set("key", d.Get("key").(string))
	d.Set("label", sshKeyReq.Label)
	d.Set("uuid", sshKeyReq.Uuid)
	d.Set("comment", sshKeyReq.Comment)

	return nil
}

func resourceSshKeysUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	sshKey := expandsshKey(d)

	sshKeyBody := &bitbucket.SshApiUsersSelectedUserSshKeysKeyIdPutOpts{
		Body: optional.NewInterface(sshKey),
	}

	user, keyId, err := sshKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, res, err := sshApi.UsersSelectedUserSshKeysKeyIdPut(c.AuthContext, keyId, user, sshKeyBody)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return resourceSshKeysRead(ctx, d, m)
}

func resourceSshKeysDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	user, keyId, err := sshKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := sshApi.UsersSelectedUserSshKeysKeyIdDelete(c.AuthContext, keyId, user)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func expandsshKey(d *schema.ResourceData) *bitbucket.SshAccountKey {
	key := &bitbucket.SshAccountKey{
		Key:   d.Get("key").(string),
		Label: d.Get("label").(string),
	}

	return key
}

func sshKeyId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected USER-ID/KEY-ID", id)
	}

	return parts[0], parts[1], nil
}
