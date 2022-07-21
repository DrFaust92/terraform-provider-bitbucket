package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/migara/bitbucket-go-client"
	"github.com/antihax/optional"
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
		Create: resourceSshKeysCreate,
		Read:   resourceSshKeysRead,
		Update: resourceSshKeysUpdate,
		Delete: resourceSshKeysDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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

func resourceSshKeysCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	sshKey := expandsshKey(d)

	sshKeyBody := &bitbucket.SshApiUsersSelectedUserSshKeysPostOpts{
		Body: optional.NewInterface(sshKey),
	}

	user := d.Get("user").(string)
	sshKeyReq, _, err := sshApi.UsersSelectedUserSshKeysPost(c.AuthContext, user, sshKeyBody)
	if err != nil {
		return fmt.Errorf("error creating ssh key: %w", err)
	}

	d.SetId(string(fmt.Sprintf("%s/%s", user, sshKeyReq.Uuid)))

	return resourceSshKeysRead(d, m)
}

func resourceSshKeysRead(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	user, keyId, err := sshKeyId(d.Id())
	if err != nil {
		return err
	}

	sshKeyReq, res, err := sshApi.UsersSelectedUserSshKeysKeyIdGet(c.AuthContext, keyId, user)
	if err != nil {
		return fmt.Errorf("error reading ssh key (%s): %w", d.Id(), err)
	}

	if res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] SSH Key (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if res.Body == nil {
		return fmt.Errorf("error getting SSH Key (%s): empty response", d.Id())
	}

	d.Set("user", user)
	d.Set("key", d.Get("key").(string))
	d.Set("label", sshKeyReq.Label)
	d.Set("uuid", sshKeyReq.Uuid)
	d.Set("comment", sshKeyReq.Comment)

	return nil
}

func resourceSshKeysUpdate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	sshKey := expandsshKey(d)

	sshKeyBody := &bitbucket.SshApiUsersSelectedUserSshKeysKeyIdPutOpts{
		Body: optional.NewInterface(sshKey),
	}

	user, keyId, err := sshKeyId(d.Id())
	if err != nil {
		return err
	}

	_, _, err = sshApi.UsersSelectedUserSshKeysKeyIdPut(c.AuthContext, keyId, user, sshKeyBody)
	if err != nil {
		return fmt.Errorf("error updating ssh key (%s): %w", d.Id(), err)
	}

	return resourceSshKeysRead(d, m)
}

func resourceSshKeysDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	sshApi := c.ApiClient.SshApi

	user, keyId, err := sshKeyId(d.Id())
	if err != nil {
		return err
	}

	res, err := sshApi.UsersSelectedUserSshKeysKeyIdDelete(c.AuthContext, keyId, user)
	if err != nil {
		if res.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("error deleting ssh key (%s): %w", d.Id(), err)
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
