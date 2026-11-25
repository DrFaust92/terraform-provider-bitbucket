package bitbucket

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	oauth2bitbucket "golang.org/x/oauth2/bitbucket"
	oauth2clientcreds "golang.org/x/oauth2/clientcredentials"
)

type ProviderConfig struct {
	ApiClient   *bitbucket.APIClient
	AuthContext context.Context
}

type Clients struct {
	genClient  ProviderConfig
	httpClient Client
}

// Provider will create the necessary terraform provider to talk to the
// Bitbucket APIs you should either specify Username and App Password, OAuth
// Client Credentials or a valid OAuth Access Token.
//
// See the Bitbucket authentication documentation for more:
// https://developer.atlassian.com/cloud/bitbucket/rest/intro/#authentication
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Optional:      true,
				Type:          schema.TypeString,
				DefaultFunc:   schema.EnvDefaultFunc("BITBUCKET_USERNAME", nil),
				ConflictsWith: []string{"oauth_client_id", "oauth_client_secret", "oauth_token"},
				RequiredWith:  []string{"password"},
			},
			"password": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("BITBUCKET_PASSWORD", nil),
				ConflictsWith: []string{"oauth_client_id", "oauth_client_secret", "oauth_token"},
				RequiredWith:  []string{"username"},
			},
			"oauth_client_id": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("BITBUCKET_OAUTH_CLIENT_ID", nil),
				ConflictsWith: []string{"username", "password", "oauth_token"},
				RequiredWith:  []string{"oauth_client_secret"},
			},
			"oauth_client_secret": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.EnvDefaultFunc("BITBUCKET_OAUTH_CLIENT_SECRET", nil),
				ConflictsWith: []string{"username", "password", "oauth_token"},
				RequiredWith:  []string{"oauth_client_id"},
			},
			"oauth_token": {
				Type:          schema.TypeString,
				Optional:      true,
				DefaultFunc:   schema.MultiEnvDefaultFunc([]string{"BITBUCKET_OAUTH_TOKEN", "BITBUCKET_ACCESS_TOKEN"}, nil),
				ConflictsWith: []string{"username", "password", "oauth_client_id", "oauth_client_secret"},
			},
			"api_url": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("BITBUCKET_API_URL", BitbucketEndpoint),
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
				Description:  "Base URL of the Bitbucket Cloud API, for example to go through a proxy or an API mock. Can also be set with the `BITBUCKET_API_URL` environment variable. Defaults to `" + BitbucketEndpoint + "`.",
			},
			"timeout": {
				Type:         schema.TypeString,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("BITBUCKET_TIMEOUT", defaultHTTPTimeout.String()),
				ValidateFunc: validatePositiveDuration,
				Description:  "Timeout of each API request, as a duration such as `30s` or `5m`. Can also be set with the `BITBUCKET_TIMEOUT` environment variable. Defaults to `2m`.",
			},
			"max_retries": {
				Type:         schema.TypeInt,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("BITBUCKET_MAX_RETRIES", defaultMaxRetries),
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "How many times to retry a request that Bitbucket rate limits (HTTP 429). Can also be set with the `BITBUCKET_MAX_RETRIES` environment variable. Defaults to `5`.",
			},
		},
		ConfigureFunc: providerConfigure,
		ResourcesMap: map[string]*schema.Resource{
			"bitbucket_branch_restriction":          resourceBranchRestriction(),
			"bitbucket_branching_model":             resourceBranchingModel(),
			"bitbucket_commit_file":                 resourceCommitFile(),
			"bitbucket_default_reviewers":           resourceDefaultReviewers(),
			"bitbucket_deploy_key":                  resourceDeployKey(),
			"bitbucket_deployment":                  resourceDeployment(),
			"bitbucket_deployment_variable":         resourceDeploymentVariable(),
			"bitbucket_forked_repository":           resourceForkedRepository(),
			"bitbucket_gpg_key":                     resourceGpgKey(),
			"bitbucket_group":                       resourceGroup(),
			"bitbucket_group_membership":            resourceGroupMembership(),
			"bitbucket_hook":                        resourceHook(),
			"bitbucket_pipeline_schedule":           resourcePipelineSchedule(),
			"bitbucket_pipeline_ssh_key":            resourcePipelineSshKey(),
			"bitbucket_pipeline_runner":             resourcePipelineRunner(),
			"bitbucket_pipeline_ssh_known_host":     resourcePipelineSshKnownHost(),
			"bitbucket_project":                     resourceProject(),
			"bitbucket_project_branching_model":     resourceProjectBranchingModel(),
			"bitbucket_project_default_reviewers":   resourceProjectDefaultReviewers(),
			"bitbucket_project_group_permission":    resourceProjectGroupPermission(),
			"bitbucket_project_user_permission":     resourceProjectUserPermission(),
			"bitbucket_repository":                  resourceRepository(),
			"bitbucket_repository_group_permission": resourceRepositoryGroupPermission(),
			"bitbucket_repository_user_permission":  resourceRepositoryUserPermission(),
			"bitbucket_repository_variable":         resourceRepositoryVariable(),
			"bitbucket_ssh_key":                     resourceSshKey(),
			"bitbucket_workspace_hook":              resourceWorkspaceHook(),
			"bitbucket_workspace_variable":          resourceWorkspaceVariable(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"bitbucket_current_user":              dataCurrentUser(),
			"bitbucket_deployment":                dataDeployment(),
			"bitbucket_deployments":               dataDeployments(),
			"bitbucket_file":                      dataFile(),
			"bitbucket_group":                     dataGroup(),
			"bitbucket_group_members":             dataGroupMembers(),
			"bitbucket_groups":                    dataGroups(),
			"bitbucket_hook_types":                dataHookTypes(),
			"bitbucket_ip_ranges":                 dataIPRanges(),
			"bitbucket_pipeline_oidc_config":      dataPipelineOidcConfig(),
			"bitbucket_pipeline_oidc_config_keys": dataPipelineOidcConfigKeys(),
			"bitbucket_project":                   dataProject(),
			"bitbucket_repository":                dataRepository(),
			"bitbucket_user":                      dataUser(),
			"bitbucket_workspace":                 dataWorkspace(),
			"bitbucket_workspace_members":         dataWorkspaceMembers(),
		},
	}
}

const (
	// defaultHTTPTimeout bounds each API request so an unresponsive endpoint
	// cannot hang a plan or apply indefinitely.
	defaultHTTPTimeout = 2 * time.Minute
	// defaultMaxRetries is how often a rate-limited request is retried.
	defaultMaxRetries = 5
)

func newHTTPClient(timeout time.Duration, maxRetries int) *http.Client {
	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{Timeout: timeout}
	client.Logger = nil
	client.RetryMax = maxRetries
	client.RetryWaitMax = 10 * time.Second
	client.RequestLogHook = func(_ retryablehttp.Logger, req *http.Request, attempt int) {
		if attempt > 0 {
			log.Printf("[DEBUG] Bitbucket rate limited, retrying %s %s (attempt %d)", req.Method, req.URL.Path, attempt)
		}
	}
	client.CheckRetry = func(ctx context.Context, response *http.Response, err error) (bool, error) {
		if err != nil {
			return false, err
		}
		return response.StatusCode == http.StatusTooManyRequests, nil
	}
	client.Backoff = func(waitMin, waitMax time.Duration, attempt int, resp *http.Response) time.Duration {
		return min(retryablehttp.RateLimitLinearJitterBackoff(waitMin, waitMax, attempt, resp), waitMax)
	}
	client.ErrorHandler = retryablehttp.PassthroughErrorHandler
	return client.StandardClient()
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	authCtx := context.Background()

	timeout, err := time.ParseDuration(d.Get("timeout").(string))
	if err != nil {
		return nil, fmt.Errorf("timeout: %w", err)
	}
	baseURL := d.Get("api_url").(string)
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	httpClient := newHTTPClient(timeout, d.Get("max_retries").(int))
	client := &Client{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
	}

	if username, ok := d.GetOk("username"); ok {
		var password interface{}
		if password, ok = d.GetOk("password"); !ok {
			return nil, fmt.Errorf("found username for basic auth, but password not specified")
		}
		log.Printf("[DEBUG] Using API Basic Auth")

		user := username.(string)
		pass := password.(string)

		cred := bitbucket.BasicAuth{
			UserName: user,
			Password: pass,
		}
		authCtx = context.WithValue(authCtx, bitbucket.ContextBasicAuth, cred)
		client.Username = &user
		client.Password = &pass
	}

	if v, ok := d.GetOk("oauth_token"); ok && v.(string) != "" {
		token := v.(string)
		client.OAuthToken = &token
		authCtx = context.WithValue(authCtx, bitbucket.ContextAccessToken, token)
	}

	if clientID, ok := d.GetOk("oauth_client_id"); ok {
		clientSecret, ok := d.GetOk("oauth_client_secret")
		if !ok {
			return nil, fmt.Errorf("found client ID for OAuth via Client Credentials Grant, but client secret was not specified")
		}

		config := &oauth2clientcreds.Config{
			ClientID:     clientID.(string),
			ClientSecret: clientSecret.(string),
			TokenURL:     oauth2bitbucket.Endpoint.TokenURL,
		}

		tokenSource := config.TokenSource(authCtx)

		client.OAuthTokenSource = tokenSource
		authCtx = context.WithValue(authCtx, bitbucket.ContextOAuth2, tokenSource)
	}

	conf := bitbucket.NewConfiguration()
	conf.HTTPClient = httpClient
	conf.BasePath = baseURL + "2.0"
	apiClient := ProviderConfig{
		ApiClient:   bitbucket.NewAPIClient(conf),
		AuthContext: authCtx,
	}

	clients := Clients{
		genClient:  apiClient,
		httpClient: *client,
	}

	return clients, nil
}

// validatePositiveDuration accepts a positive Go duration such as "30s" or "5m".
func validatePositiveDuration(v interface{}, k string) ([]string, []error) {
	d, err := time.ParseDuration(v.(string))
	if err != nil {
		return nil, []error{fmt.Errorf("%q must be a duration such as 30s or 5m: %w", k, err)}
	}
	if d <= 0 {
		return nil, []error{fmt.Errorf("%q must be positive", k)}
	}
	return nil, nil
}
