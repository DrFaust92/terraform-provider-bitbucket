package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Hook is the hook you want to add to a bitbucket repository
type Hook struct {
	UUID                 string   `json:"uuid,omitempty"`
	URL                  string   `json:"url,omitempty"`
	Description          string   `json:"description,omitempty"`
	Active               bool     `json:"active"`
	SkipCertVerification bool     `json:"skip_cert_verification"`
	Events               []string `json:"events,omitempty"`
	Secret               string   `json:"secret"`
	SecretSet            bool     `json:"secret_set,omitempty"`
	HistoryEnabled       bool     `json:"history_enabled,omitempty"`
}

func resourceHook() *schema.Resource {
	secretWO, secretWOVersion := writeOnlySchemas("secret")

	return &schema.Resource{
		CreateWithoutTimeout: resourceHookCreate,
		ReadWithoutTimeout:   resourceHookRead,
		UpdateWithoutTimeout: resourceHookUpdate,
		DeleteWithoutTimeout: resourceHookDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
					return nil, fmt.Errorf("unexpected format of ID (%q), expected OWNER/REPO/HOOK-ID", d.Id())
				}
				d.SetId(idParts[2])
				d.Set("owner", idParts[0])
				d.Set("repository", idParts[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"owner": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"secret_set": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"history_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"url": {
				Type:     schema.TypeString,
				Required: true,
			},
			"secret": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"secret_wo":         secretWO,
			"secret_wo_version": secretWOVersion,
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"events": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"issue:comment_created",
						"issue:created",
						"issue:updated",
						"project:updated",
						"pullrequest:approved",
						"pullrequest:changes_request_created",
						"pullrequest:changes_request_removed",
						"pullrequest:comment_created",
						"pullrequest:comment_deleted",
						"pullrequest:comment_reopened",
						"pullrequest:comment_resolved",
						"pullrequest:comment_updated",
						"pullrequest:created",
						"pullrequest:fulfilled",
						"pullrequest:rejected",
						"pullrequest:unapproved",
						"pullrequest:updated",
						"repo:commit_comment_created",
						"repo:commit_status_created",
						"repo:commit_status_updated",
						"repo:created",
						"repo:deleted",
						"repo:fork",
						"repo:imported",
						"repo:push",
						"repo:transfer",
						"repo:updated",
					}, false),
				},
			},
			"skip_cert_verification": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func createHook(d *schema.ResourceData) *Hook {

	events := make([]string, 0, len(d.Get("events").(*schema.Set).List()))

	for _, item := range d.Get("events").(*schema.Set).List() {
		events = append(events, item.(string))
	}

	hook := &Hook{
		URL:                  d.Get("url").(string),
		Description:          d.Get("description").(string),
		Active:               d.Get("active").(bool),
		HistoryEnabled:       d.Get("history_enabled").(bool),
		SkipCertVerification: d.Get("skip_cert_verification").(bool),
		Secret:               writeOnlyOrValue(d, "secret"),
		Events:               events,
	}

	return hook
}

func resourceHookCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	hook := createHook(d)

	payload, err := json.Marshal(hook)
	if err != nil {
		return diag.FromErr(err)
	}

	hookReq, err := client.Post(fmt.Sprintf("2.0/repositories/%s/%s/hooks",
		d.Get("owner").(string),
		d.Get("repository").(string),
	), bytes.NewBuffer(payload))

	if err != nil {
		return diag.FromErr(err)
	}

	body, readerr := io.ReadAll(hookReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	decodeerr := json.Unmarshal(body, &hook)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	d.SetId(hook.UUID)

	return resourceHookRead(ctx, d, m)
}
func resourceHookRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	hookReq, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
		d.Get("owner").(string),
		d.Get("repository").(string),
		url.PathEscape(d.Id()),
	))

	if hookReq != nil && hookReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository Hook (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("ID: %s", url.PathEscape(d.Id()))

	if hookReq.StatusCode == 200 {
		var hook Hook

		body, readerr := io.ReadAll(hookReq.Body)
		if readerr != nil {
			return diag.FromErr(readerr)
		}

		log.Printf("[DEBUG] Hook json: %s", string(body))

		decodeerr := json.Unmarshal(body, &hook)
		if decodeerr != nil {
			return diag.FromErr(decodeerr)
		}

		d.Set("uuid", hook.UUID)
		d.Set("description", hook.Description)
		d.Set("active", hook.Active)
		d.Set("history_enabled", hook.HistoryEnabled)
		d.Set("secret_set", hook.SecretSet)
		d.Set("url", hook.URL)
		d.Set("secret", d.Get("secret").(string))
		d.Set("skip_cert_verification", hook.SkipCertVerification)
		d.Set("events", hook.Events)
	}

	return nil
}

func resourceHookUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	hook := createHook(d)
	payload, err := json.Marshal(hook)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
		d.Get("owner").(string),
		d.Get("repository").(string),
		url.PathEscape(d.Id()),
	), bytes.NewBuffer(payload))

	if err != nil {
		return diag.FromErr(err)
	}

	return resourceHookRead(ctx, d, m)
}

func resourceHookDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	_, err := client.Delete(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s",
		d.Get("owner").(string),
		d.Get("repository").(string),
		url.PathEscape(d.Id()),
	))

	return diag.FromErr(err)

}
