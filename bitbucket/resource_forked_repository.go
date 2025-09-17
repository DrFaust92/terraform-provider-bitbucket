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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceForkedRepository() *schema.Resource {
	return &schema.Resource{
		CreateContext:        resourceForkedRepositoryCreate,
		UpdateWithoutTimeout: resourceRepositoryUpdate,
		ReadContext:          resourceForkedRepositoryRead,
		DeleteWithoutTimeout: resourceRepositoryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"scm": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"has_wiki": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"has_issues": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"website": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"clone_ssh": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"clone_https": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"project_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"is_private": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"pipelines_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"fork_policy": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "allow_forks",
				ValidateFunc: validation.StringInSlice([]string{"allow_forks", "no_public_forks", "no_forks"}, false),
			},
			"language": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return computeSlug(old) == computeSlug(new)
				},
			},
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"link": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"avatar": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"href": {
										Type:     schema.TypeString,
										Optional: true,
										DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
											return strings.HasPrefix(old, "https://bytebucket.org/ravatar/")
										},
									},
								},
							},
						},
					},
				},
			},
			"parent": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required: true,
				ForceNew: true,
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(map[string]interface{})
					if _, ok := v["slug"]; !ok {
						errs = append(errs, fmt.Errorf("a repository 'slug' is required when specifying a parent to fork from"))
					}
					if _, ok := v["owner"]; !ok {
						errs = append(errs, fmt.Errorf("a repository 'owner' is required when specifying a parent to fork from"))
					}
					return warns, errs
				},
			},
		},
	}
}

type forkWorkspace struct {
	Slug string `json:"slug,omitempty"`
}

type forkedRepositoryBody struct {
	Name        string                     `json:"name,omitempty"`
	Language    string                     `json:"language,omitempty"`
	IsPrivate   bool                       `json:"is_private,omitempty"`
	Description string                     `json:"description,omitempty"`
	ForkPolicy  string                     `json:"fork_policy,omitempty"`
	HasWiki     bool                       `json:"has_wiki,omitempty"`
	HasIssues   bool                       `json:"has_issues,omitempty"`
	Links       *bitbucket.RepositoryLinks `json:"links,omitempty"`
	Project     *bitbucket.Project         `json:"project,omitempty"`
	Workspace   *forkWorkspace             `json:"workspace,omitempty"`
}

func createForkedRepositoryFromRepository(repo *bitbucket.Repository, targetWorkspaceSlug string) *forkedRepositoryBody {
	forkedRepo := &forkedRepositoryBody{
		Name:        repo.Name,
		Language:    repo.Language,
		IsPrivate:   repo.IsPrivate,
		Description: repo.Description,
		ForkPolicy:  repo.ForkPolicy,
		HasWiki:     repo.HasWiki,
		HasIssues:   repo.HasIssues,
	}

	workspace := &forkWorkspace{
		Slug: targetWorkspaceSlug,
	}
	forkedRepo.Workspace = workspace
	if repo.Links != nil {
		forkedRepo.Links = repo.Links
	}
	if repo.Project != nil {
		forkedRepo.Project = repo.Project
	}
	return forkedRepo
}

func resourceForkedRepositoryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	repoApi := c.ApiClient.RepositoriesApi
	pipeApi := c.ApiClient.PipelinesApi
	repo := newRepositoryFromResource(d)

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}
	repoSlug = computeSlug(repoSlug)

	workspace := d.Get("owner").(string)
	parent := d.Get("parent").(map[string]interface{})
	parentRepoSlug := parent["slug"].(string)
	parentWorkspace := parent["owner"].(string)
	requestRepo := createForkedRepositoryFromRepository(repo, workspace)
	repoBody := &bitbucket.RepositoriesApiRepositoriesWorkspaceRepoSlugForksPostOpts{
		Body: optional.NewInterface(requestRepo),
	}
	_, res, err := repoApi.RepositoriesWorkspaceRepoSlugForksPost(c.AuthContext, parentRepoSlug, parentWorkspace, repoBody)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", d.Get("owner").(string), repoSlug))

	pipelinesEnabled := d.Get("pipelines_enabled").(bool)
	pipelinesConfig := &bitbucket.PipelinesConfig{Enabled: pipelinesEnabled}

	//nolint:all
	retryErr := resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		_, pipelineResponse, err := pipeApi.UpdateRepositoryPipelineConfig(c.AuthContext, *pipelinesConfig, workspace, repoSlug)
		if pipelineResponse.StatusCode == 403 || pipelineResponse.StatusCode == 404 {
			return resource.RetryableError(
				fmt.Errorf("permissions error setting Pipelines config, retrying"),
			)
		}

		if err := handleClientError(pipelineResponse, err); err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if retryErr != nil {
		return diag.FromErr(retryErr)
	}

	return resourceRepositoryRead(ctx, d, m)
}

func resourceForkedRepositoryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	id := d.Id()
	if id != "" {
		idparts := strings.Split(id, "/")
		if len(idparts) == 2 {
			d.Set("owner", idparts[0])
			d.Set("slug", idparts[1])
		} else {
			return diag.Errorf("incorrect ID format, should match `owner/slug`")
		}
	}

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}
	repoSlug = computeSlug(repoSlug)

	workspace := d.Get("owner").(string)
	c := m.(Clients).genClient
	repoApi := c.ApiClient.RepositoriesApi
	pipeApi := c.ApiClient.PipelinesApi

	repoRes, res, err := repoApi.RepositoriesWorkspaceRepoSlugGet(c.AuthContext, repoSlug, workspace)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("scm", repoRes.Scm)
	d.Set("is_private", repoRes.IsPrivate)
	d.Set("has_wiki", repoRes.HasWiki)
	d.Set("has_issues", repoRes.HasIssues)
	d.Set("name", repoRes.Name)
	d.Set("slug", repoRes.Name)
	d.Set("language", repoRes.Language)
	d.Set("fork_policy", repoRes.ForkPolicy)
	// d.Set("website", repoRes.Website)
	d.Set("description", repoRes.Description)
	d.Set("project_key", repoRes.Project.Key)
	d.Set("uuid", repoRes.Uuid)

	if repoRes.Parent != nil {
		parentMap := make(map[string]string)
		parentOwner, parentSlug, splitErr := splitFullName(repoRes.Parent.FullName)
		if splitErr != nil {
			return diag.Errorf("error reading forked repository (%s)", d.Get("name").(string))
		}
		parentMap["owner"] = parentOwner
		parentMap["slug"] = parentSlug
		d.Set("parent", parentMap)
	}

	for _, cloneURL := range repoRes.Links.Clone {
		if cloneURL.Name == "https" {
			d.Set("clone_https", cloneURL.Href)
		} else {
			d.Set("clone_ssh", cloneURL.Href)
		}
	}

	d.Set("link", flattenLinks(repoRes.Links))

	pipelinesConfigReq, res, err := pipeApi.GetRepositoryPipelineConfig(c.AuthContext, workspace, repoSlug)
	if err := handleClientError(res, err); err != nil && res.StatusCode != http.StatusNotFound {
		return diag.FromErr(err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		d.Set("pipelines_enabled", pipelinesConfigReq.Enabled)
	case http.StatusNotFound:
		d.Set("pipelines_enabled", false)
	}

	return nil
}
