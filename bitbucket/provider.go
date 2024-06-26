package bitbucket

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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
				DefaultFunc:   schema.EnvDefaultFunc("BITBUCKET_OAUTH_TOKEN", nil),
				ConflictsWith: []string{"username", "password", "oauth_client_id", "oauth_client_secret"},
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
			"bitbucket_group":                       resourceGroup(),
			"bitbucket_group_membership":            resourceGroupMembership(),
			"bitbucket_hook":                        resourceHook(),
			"bitbucket_pipeline_schedule":           resourcePipelineSchedule(),
			"bitbucket_pipeline_ssh_key":            resourcePipelineSshKey(),
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
			"bitbucket_group":                     dataGroup(),
			"bitbucket_group_members":             dataGroupMembers(),
			"bitbucket_groups":                    dataGroups(),
			"bitbucket_hook_types":                dataHookTypes(),
			"bitbucket_ip_ranges":                 dataIPRanges(),
			"bitbucket_pipeline_oidc_config":      dataPipelineOidcConfig(),
			"bitbucket_pipeline_oidc_config_keys": dataPipelineOidcConfigKeys(),
			"bitbucket_repository":                dataRepository(),
			"bitbucket_user":                      dataUser(),
			"bitbucket_workspace":                 dataWorkspace(),
			"bitbucket_workspace_members":         dataWorkspaceMembers(),
		},
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	authCtx := context.Background()

	client := &Client{
		HTTPClient: &http.Client{},
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
