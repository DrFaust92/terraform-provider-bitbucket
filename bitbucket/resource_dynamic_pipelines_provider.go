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

// This resource talks to Bitbucket Cloud's undocumented /internal API.
// REST 2.0 does not currently expose the Dynamic Pipelines provider selector.
// Atlassian may change or remove this endpoint without notice.
const (
	dppProviderNotConfiguredKey = "account-service.dynamic-pipelines.provider-not-configured"
)

type dynamicPipelinesProviderConfig struct {
	AppAri          string `json:"appAri"`
	FallbackEnabled bool   `json:"fallbackEnabled"`
}

func resourceDynamicPipelinesProvider() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceDynamicPipelinesProviderCreate,
		ReadWithoutTimeout:   resourceDynamicPipelinesProviderRead,
		UpdateWithoutTimeout: resourceDynamicPipelinesProviderUpdate,
		DeleteWithoutTimeout: resourceDynamicPipelinesProviderDelete,
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
			"app_ari": {
				Type:     schema.TypeString,
				Required: true,
			},
			"fallback_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}

func resourceDynamicPipelinesProviderCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	workspace := d.Get("workspace").(string)
	repository := d.Get("repository").(string)
	desired := d.Get("app_ari").(string)

	remote, notConfigured, err := fetchDynamicPipelinesProvider(client, workspace, repository)
	if err != nil {
		return diag.FromErr(err)
	}

	if !notConfigured {
		if remote.AppAri == desired {
			d.SetId(fmt.Sprintf("%s/%s", workspace, repository))
			return setDynamicPipelinesProviderState(d, workspace, repository, remote)
		}

		return diag.Errorf(
			"dynamic pipelines provider for %s/%s is already configured with appAri %q; refusing to overwrite with %q",
			workspace, repository, remote.AppAri, desired,
		)
	}

	if err := postDynamicPipelinesProvider(client, workspace, repository, desired); err != nil {
		return diag.FromErr(err)
	}

	verified, err := verifyDynamicPipelinesProvider(client, workspace, repository, desired)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", workspace, repository))
	return setDynamicPipelinesProviderState(d, workspace, repository, verified)
}

func resourceDynamicPipelinesProviderRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, repository, err := dppId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	remote, notConfigured, err := fetchDynamicPipelinesProvider(client, workspace, repository)
	if err != nil {
		return diag.FromErr(err)
	}

	if notConfigured {
		log.Printf("[WARN] Dynamic Pipelines provider (%s) is not configured, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	return setDynamicPipelinesProviderState(d, workspace, repository, remote)
}

func resourceDynamicPipelinesProviderUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	workspace, repository, err := dppId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	desired := d.Get("app_ari").(string)

	remote, notConfigured, err := fetchDynamicPipelinesProvider(client, workspace, repository)
	if err != nil {
		return diag.FromErr(err)
	}

	if notConfigured {
		return diag.Errorf("dynamic pipelines provider for %s/%s is not configured; cannot update", workspace, repository)
	}

	if remote.AppAri == desired {
		return setDynamicPipelinesProviderState(d, workspace, repository, remote)
	}

	if err := postDynamicPipelinesProvider(client, workspace, repository, desired); err != nil {
		return diag.FromErr(err)
	}

	verified, err := verifyDynamicPipelinesProvider(client, workspace, repository, desired)
	if err != nil {
		return diag.FromErr(err)
	}

	return setDynamicPipelinesProviderState(d, workspace, repository, verified)
}

// Delete is state-only. Bitbucket has no documented DELETE for this selector,
// and posting an empty appAri is untested. Terraform drops the resource from
// state and leaves the remote configuration intact.
func resourceDynamicPipelinesProviderDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func fetchDynamicPipelinesProvider(client Client, workspace, repository string) (*dynamicPipelinesProviderConfig, bool, error) {
	resp, err := client.Get(dynamicPipelinesProviderEndpoint(workspace, repository))
	if err != nil {
		if isDynamicPipelinesProviderNotConfigured(err) {
			return nil, true, nil
		}
		return nil, false, err
	}

	if resp == nil || resp.Body == nil {
		return nil, false, fmt.Errorf("malformed Dynamic Pipelines provider response for %s/%s: empty body", workspace, repository)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("reading Dynamic Pipelines provider response for %s/%s: %w", workspace, repository, err)
	}

	var cfg dynamicPipelinesProviderConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, false, fmt.Errorf("malformed Dynamic Pipelines provider response for %s/%s: %w", workspace, repository, err)
	}

	if cfg.AppAri == "" {
		return nil, false, fmt.Errorf("malformed Dynamic Pipelines provider response for %s/%s: missing appAri", workspace, repository)
	}

	return &cfg, false, nil
}

func postDynamicPipelinesProvider(client Client, workspace, repository, appAri string) error {
	payload, err := json.Marshal(map[string]string{"appAri": appAri})
	if err != nil {
		return err
	}

	resp, err := client.Post(dynamicPipelinesProviderEndpoint(workspace, repository), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return nil
}

func verifyDynamicPipelinesProvider(client Client, workspace, repository, desired string) (*dynamicPipelinesProviderConfig, error) {
	remote, notConfigured, err := fetchDynamicPipelinesProvider(client, workspace, repository)
	if err != nil {
		return nil, err
	}
	if notConfigured {
		return nil, fmt.Errorf("dynamic pipelines provider verification failed for %s/%s: remote is not configured after apply", workspace, repository)
	}
	if remote.AppAri != desired {
		return nil, fmt.Errorf("dynamic pipelines provider verification failed for %s/%s: remote appAri %q does not match desired %q", workspace, repository, remote.AppAri, desired)
	}
	return remote, nil
}

func setDynamicPipelinesProviderState(d *schema.ResourceData, workspace, repository string, cfg *dynamicPipelinesProviderConfig) diag.Diagnostics {
	d.Set("workspace", workspace)
	d.Set("repository", repository)
	d.Set("app_ari", cfg.AppAri)
	d.Set("fallback_enabled", cfg.FallbackEnabled)
	return nil
}

func isDynamicPipelinesProviderNotConfigured(err error) bool {
	apiErr, ok := err.(Error)
	if !ok {
		return false
	}
	if apiErr.StatusCode != http.StatusNotFound {
		return false
	}
	return apiErr.Key == dppProviderNotConfiguredKey || apiErr.APIError.Key == dppProviderNotConfiguredKey
}

func dynamicPipelinesProviderEndpoint(workspace, repository string) string {
	return fmt.Sprintf("internal/repositories/%s/%s/pipelines-config/dynamic-pipelines-provider", workspace, repository)
}

func dppId(id string) (string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE/REPOSITORY", id)
	}
	return parts[0], parts[1], nil
}
