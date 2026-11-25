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
)

// pipelineRunner is the runner object of the Bitbucket Pipelines runners API.
// The generated client's create and update methods send no request body, so
// the raw HTTP client is used for all runner requests.
type pipelineRunner struct {
	UUID        string                     `json:"uuid,omitempty"`
	Name        string                     `json:"name,omitempty"`
	Labels      []string                   `json:"labels,omitempty"`
	State       *pipelineRunnerState       `json:"state,omitempty"`
	OAuthClient *pipelineRunnerOAuthClient `json:"oauth_client,omitempty"`
}

type pipelineRunnerState struct {
	Status   string `json:"status,omitempty"`
	Cordoned bool   `json:"cordoned,omitempty"`
}

type pipelineRunnerOAuthClient struct {
	ID            string `json:"id,omitempty"`
	Secret        string `json:"secret,omitempty"`
	TokenEndpoint string `json:"token_endpoint,omitempty"`
	Audience      string `json:"audience,omitempty"`
}

func resourcePipelineRunner() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourcePipelineRunnerCreate,
		ReadWithoutTimeout:   resourcePipelineRunnerRead,
		UpdateWithoutTimeout: resourcePipelineRunnerUpdate,
		DeleteWithoutTimeout: resourcePipelineRunnerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcePipelineRunnerImport,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The workspace of the runner.",
			},
			"repository": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The repository slug for a repository runner. Omit it for a workspace runner.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the runner.",
			},
			"labels": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Labels that pipeline steps use to select the runner, for example `self.hosted` and `linux`.",
			},
			"uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the runner.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the runner, for example `ONLINE` or `UNREGISTERED`.",
			},
			"cordoned": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the runner is prevented from accepting new steps.",
			},
			"oauth_client_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The OAuth client ID the runner authenticates with.",
			},
			"oauth_client_secret": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The OAuth client secret the runner authenticates with. Bitbucket returns it only when the runner is created, so it is empty after an import.",
			},
			"oauth_token_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The OAuth token endpoint the runner uses.",
			},
			"oauth_audience": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The audience of the runner's OAuth tokens.",
			},
		},
	}
}

// pipelineRunnersEndpoint returns the API path of a workspace's or a
// repository's runners, or of one runner when uuid is set.
func pipelineRunnersEndpoint(workspace, repository, uuid string) string {
	endpoint := fmt.Sprintf("2.0/workspaces/%s/pipelines-config/runners", url.PathEscape(workspace))
	if repository != "" {
		endpoint = fmt.Sprintf("2.0/repositories/%s/%s/pipelines-config/runners", url.PathEscape(workspace), url.PathEscape(repository))
	}
	if uuid != "" {
		endpoint += "/" + url.PathEscape(uuid)
	}
	return endpoint
}

func expandPipelineRunner(d *schema.ResourceData) *pipelineRunner {
	labels := make([]string, 0, d.Get("labels").(*schema.Set).Len())
	for _, l := range d.Get("labels").(*schema.Set).List() {
		labels = append(labels, l.(string))
	}
	return &pipelineRunner{Name: d.Get("name").(string), Labels: labels}
}

// doPipelineRunnerRequest sends a runner request and decodes the response
// into out, when out is not nil.
func doPipelineRunnerRequest(client Client, method, endpoint string, body *pipelineRunner, out *pipelineRunner) (*http.Response, error) {
	var payload *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		payload = bytes.NewBuffer(data)
	}

	res, err := client.Do(method, endpoint, payload, "application/json")
	if err != nil {
		return res, err
	}
	defer res.Body.Close()

	if out == nil {
		return res, nil
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return res, err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return res, fmt.Errorf("decoding runner response: %w", err)
	}
	return res, nil
}

func resourcePipelineRunnerCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	workspace := d.Get("workspace").(string)
	repository := d.Get("repository").(string)

	var runner pipelineRunner
	if _, err := doPipelineRunnerRequest(client, http.MethodPost, pipelineRunnersEndpoint(workspace, repository, ""), expandPipelineRunner(d), &runner); err != nil {
		return diag.Errorf("creating pipeline runner: %s", err)
	}
	if runner.UUID == "" {
		return diag.Errorf("creating pipeline runner: the response has no uuid")
	}

	d.SetId(pipelineRunnerID(workspace, repository, runner.UUID))
	// The secret is only in the create response.
	if runner.OAuthClient != nil {
		d.Set("oauth_client_secret", runner.OAuthClient.Secret)
	}

	return resourcePipelineRunnerRead(ctx, d, m)
}

func resourcePipelineRunnerRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	workspace, repository, uuid, err := parsePipelineRunnerID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var runner pipelineRunner
	res, err := doPipelineRunnerRequest(client, http.MethodGet, pipelineRunnersEndpoint(workspace, repository, uuid), nil, &runner)
	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Pipeline runner (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.Errorf("reading pipeline runner %s: %s", d.Id(), err)
	}

	d.Set("workspace", workspace)
	d.Set("repository", repository)
	d.Set("uuid", runner.UUID)
	d.Set("name", runner.Name)
	if err := d.Set("labels", runner.Labels); err != nil {
		return diag.FromErr(err)
	}
	status, cordoned := "", false
	if runner.State != nil {
		status, cordoned = runner.State.Status, runner.State.Cordoned
	}
	d.Set("status", status)
	d.Set("cordoned", cordoned)
	if runner.OAuthClient != nil {
		d.Set("oauth_client_id", runner.OAuthClient.ID)
		d.Set("oauth_token_endpoint", runner.OAuthClient.TokenEndpoint)
		d.Set("oauth_audience", runner.OAuthClient.Audience)
		// Reads don't return the secret; keep the one from create.
		if runner.OAuthClient.Secret != "" {
			d.Set("oauth_client_secret", runner.OAuthClient.Secret)
		}
	}

	return nil
}

func resourcePipelineRunnerUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	workspace, repository, uuid, err := parsePipelineRunnerID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := doPipelineRunnerRequest(client, http.MethodPut, pipelineRunnersEndpoint(workspace, repository, uuid), expandPipelineRunner(d), nil); err != nil {
		return diag.Errorf("updating pipeline runner %s: %s", d.Id(), err)
	}

	return resourcePipelineRunnerRead(ctx, d, m)
}

func resourcePipelineRunnerDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	workspace, repository, uuid, err := parsePipelineRunnerID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := doPipelineRunnerRequest(client, http.MethodDelete, pipelineRunnersEndpoint(workspace, repository, uuid), nil, nil)
	if res != nil && res.StatusCode == http.StatusNotFound {
		return nil
	}
	if err != nil {
		return diag.Errorf("deleting pipeline runner %s: %s", d.Id(), err)
	}
	return nil
}

func resourcePipelineRunnerImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	if _, _, _, err := parsePipelineRunnerID(d.Id()); err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

// pipelineRunnerID is workspace/uuid for a workspace runner and
// workspace/repository/uuid for a repository runner.
func pipelineRunnerID(workspace, repository, uuid string) string {
	if repository == "" {
		return workspace + "/" + uuid
	}
	return workspace + "/" + repository + "/" + uuid
}

func parsePipelineRunnerID(id string) (workspace, repository, uuid string, err error) {
	parts := strings.Split(id, "/")
	for _, p := range parts {
		if p == "" {
			parts = nil
			break
		}
	}
	switch len(parts) {
	case 2:
		return parts[0], "", parts[1], nil
	case 3:
		return parts[0], parts[1], parts[2], nil
	}
	return "", "", "", fmt.Errorf("unexpected pipeline runner ID %q: want workspace/runner-uuid or workspace/repository/runner-uuid", id)
}
