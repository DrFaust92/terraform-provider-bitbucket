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

func resourceGpgKey() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceGpgKeysCreate,
		ReadWithoutTimeout:   resourceGpgKeysRead,
		DeleteWithoutTimeout: resourceGpgKeysDelete,
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
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"fingerprint": {
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

func resourceGpgKeysCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	gpgApi := c.ApiClient.GPGApi

	gpgKey := expandGpgKey(d)

	gpgKeyBody := &bitbucket.GPGApiUsersSelectedUserGpgKeysPostOpts{
		Body: optional.NewInterface(gpgKey),
	}

	user := d.Get("user").(string)
	gpgKeyReq, res, err := gpgApi.UsersSelectedUserGpgKeysPost(c.AuthContext, user, gpgKeyBody)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", user, gpgKeyReq.Fingerprint))

	return resourceGpgKeysRead(ctx, d, m)
}

func resourceGpgKeysRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	gpgApi := c.ApiClient.GPGApi

	user, fingerprint, err := gpgKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	gpgKeyReq, res, err := gpgApi.UsersSelectedUserGpgKeysFingerprintGet(c.AuthContext, fingerprint, user)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] GPG Key (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	if res.Body == nil {
		return diag.Errorf("error getting GPG Key (%s): empty response", d.Id())
	}

	d.Set("user", user)
	// The API returns the normalized (armored) key, which differs from the
	// submitted value; preserve the configured key to avoid a perpetual diff.
	d.Set("key", d.Get("key").(string))
	d.Set("name", gpgKeyReq.Name)
	d.Set("fingerprint", gpgKeyReq.Fingerprint)
	d.Set("key_id", gpgKeyReq.KeyId)

	return nil
}

func resourceGpgKeysDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	gpgApi := c.ApiClient.GPGApi

	user, fingerprint, err := gpgKeyId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := gpgApi.UsersSelectedUserGpgKeysFingerprintDelete(c.AuthContext, fingerprint, user)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func expandGpgKey(d *schema.ResourceData) *bitbucket.GpgAccountKey {
	key := &bitbucket.GpgAccountKey{
		Key:  d.Get("key").(string),
		Name: d.Get("name").(string),
	}

	return key
}

func gpgKeyId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected USER-ID/FINGERPRINT", id)
	}

	return parts[0], parts[1], nil
}
