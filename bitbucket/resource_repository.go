package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceRepository() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceRepositoryCreate,
		UpdateWithoutTimeout: resourceRepositoryUpdate,
		ReadWithoutTimeout:   resourceRepositoryRead,
		DeleteWithoutTimeout: resourceRepositoryDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"scm": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "git",
				ValidateFunc: validation.StringInSlice([]string{"hg", "git"}, false),
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
			"inherit_default_merge_strategy": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"inherit_branching_model": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
		},
	}
}

type RepositoryInheritanceSettings struct {
	DefaultMergeStrategy *bool `json:"default_merge_strategy,omitempty"`
	BranchingModel       *bool `json:"branching_model,omitempty"`
}

func newRepositoryFromResource(d *schema.ResourceData) *bitbucket.Repository {
	repo := &bitbucket.Repository{
		Name:        d.Get("name").(string),
		Language:    d.Get("language").(string),
		IsPrivate:   d.Get("is_private").(bool),
		Description: d.Get("description").(string),
		ForkPolicy:  d.Get("fork_policy").(string),
		HasWiki:     d.Get("has_wiki").(bool),
		HasIssues:   d.Get("has_issues").(bool),
		Scm:         d.Get("scm").(string),
	}

	if v, ok := d.GetOk("link"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		repo.Links = expandLinks(v.([]interface{}))
	}

	if v, ok := d.GetOk("project_key"); ok && v.(string) != "" {
		project := &bitbucket.Project{
			Key: v.(string),
		}
		repo.Project = project
	}

	return repo
}

func resourceRepositoryUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	repoApi := c.ApiClient.RepositoriesApi
	pipeApi := c.ApiClient.PipelinesApi
	client := m.(Clients).httpClient

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}
	repoSlug = computeSlug(repoSlug)
	workspace := d.Get("owner").(string)

	if d.HasChangesExcept("pipelines_enabled", "inherit_default_merge_strategy", "inherit_branching_model") {
		repository := newRepositoryFromResource(d)

		repoBody := &bitbucket.RepositoriesApiRepositoriesWorkspaceRepoSlugPutOpts{
			Body: optional.NewInterface(repository),
		}
		_, res, err := repoApi.RepositoriesWorkspaceRepoSlugPut(c.AuthContext, repoSlug, workspace, repoBody)
		if err := handleClientError(res, err); err != nil {
			return diag.FromErr(err)
		}
	}

	// Check if the "owner" or "name" properties have changed. If so:
	// 1. Update the resource ID used by Terraform to track the resource
	// 2. Update the workspace and repoSlug variables used to make API calls
	if d.HasChange("owner") || d.HasChange("name") {
		_, newWorkspace := d.GetChange("owner")
		_, newRepoSlug := d.GetChange("name")
		workspace = newWorkspace.(string)
		repoSlug = newRepoSlug.(string)
		newId := fmt.Sprintf("%s/%s", workspace, repoSlug)
		log.Printf("[DEBUG] repo owner/name has changed; setting new resource ID: %s", newId)
		d.SetId(newId)
	}

	if d.HasChange("pipelines_enabled") {
		// nolint:staticcheck
		if v, ok := d.GetOkExists("pipelines_enabled"); ok {
			pipelinesConfig := &bitbucket.PipelinesConfig{Enabled: v.(bool)}

			_, res, err := pipeApi.UpdateRepositoryPipelineConfig(c.AuthContext, *pipelinesConfig, workspace, repoSlug)
			if err := handleClientError(res, err); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if d.HasChanges("inherit_default_merge_strategy", "inherit_branching_model") {
		setting := createRepositoryInheritanceSettings(d)

		log.Printf("Repository Inheritance Settings update is: %#v", setting)

		payload, err := json.Marshal(setting)
		if err != nil {
			return diag.FromErr(err)
		}

		log.Printf("Repository Inheritance Settings update encoded is: %v", string(payload))

		_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/override-settings",
			workspace,
			repoSlug,
		), bytes.NewBuffer(payload))

		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceRepositoryRead(ctx, d, m)
}

func resourceRepositoryCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	repoApi := c.ApiClient.RepositoriesApi
	pipeApi := c.ApiClient.PipelinesApi
	client := m.(Clients).httpClient

	repo := newRepositoryFromResource(d)

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}
	repoSlug = computeSlug(repoSlug)

	workspace := d.Get("owner").(string)

	repoBody := &bitbucket.RepositoriesApiRepositoriesWorkspaceRepoSlugPostOpts{
		Body: optional.NewInterface(repo),
	}

	_, res, err := repoApi.RepositoriesWorkspaceRepoSlugPost(c.AuthContext, repoSlug, workspace, repoBody)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", d.Get("owner").(string), repoSlug))

	// nolint:staticcheck
	if v, ok := d.GetOkExists("pipelines_enabled"); ok {
		pipelinesConfig := &bitbucket.PipelinesConfig{Enabled: v.(bool)}

		_, res, err := pipeApi.UpdateRepositoryPipelineConfig(c.AuthContext, *pipelinesConfig, workspace, repoSlug)
		if err := handleClientError(res, err); err != nil {
			return diag.FromErr(err)
		}
	}

	// nolint:staticcheck
	_, branchOk := d.GetOkExists("inherit_branching_model")
	// nolint:staticcheck
	_, mergeStratOk := d.GetOkExists("inherit_default_merge_strategy")

	if mergeStratOk || branchOk {
		setting := createRepositoryInheritanceSettings(d)

		payload, err := json.Marshal(setting)
		if err != nil {
			return diag.FromErr(err)
		}

		_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/override-settings",
			workspace,
			repoSlug,
		), bytes.NewBuffer(payload))

		if err != nil {
			return diag.FromErr(err)
		}

	}

	return resourceRepositoryRead(ctx, d, m)
}

func resourceRepositoryRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	repoApi := c.ApiClient.RepositoriesApi
	pipeApi := c.ApiClient.PipelinesApi
	client := m.(Clients).httpClient

	workspace, repoSlug, err := repositoryId(d.Id())
	if err != nil {
		diag.FromErr(err)
	}

	// Check if the "owner" or "name" properties have changed. If so, update the
	// the workspace and repoSlug variables used to make API calls.
	if d.HasChange("owner") || d.HasChange("name") {
		_, newWorkspace := d.GetChange("owner")
		_, newRepoSlug := d.GetChange("name")
		workspace = newWorkspace.(string)
		repoSlug = newRepoSlug.(string)
		log.Printf("[DEBUG] repo owner/name has changed to %s/%s", workspace, repoSlug)
	}

	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}
	repoSlug = computeSlug(repoSlug)

	repoRes, res, err := repoApi.RepositoriesWorkspaceRepoSlugGet(c.AuthContext, repoSlug, workspace)

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.Set("owner", workspace)
	d.Set("scm", repoRes.Scm)
	d.Set("is_private", repoRes.IsPrivate)
	d.Set("has_wiki", repoRes.HasWiki)
	d.Set("has_issues", repoRes.HasIssues)
	d.Set("name", repoRes.Name)
	d.Set("slug", repoRes.Slug)
	d.Set("language", repoRes.Language)
	d.Set("fork_policy", repoRes.ForkPolicy)
	// d.Set("website", repoRes.Website)
	d.Set("description", repoRes.Description)
	d.Set("project_key", repoRes.Project.Key)
	d.Set("uuid", repoRes.Uuid)

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

	settingReq, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/override-settings",
		workspace,
		repoSlug,
	))

	if err != nil {
		return diag.FromErr(err)
	}

	var setting RepositoryInheritanceSettings

	body, readerr := io.ReadAll(settingReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("Repository Inheritance Settings raw is: %#v", string(body))

	decodeerr := json.Unmarshal(body, &setting)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("Repository Inheritance Settings decoded is: %#v", setting)

	d.Set("inherit_default_merge_strategy", setting.DefaultMergeStrategy)
	d.Set("inherit_branching_model", setting.BranchingModel)

	return nil
}

func resourceRepositoryDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	var repoSlug string
	repoSlug = d.Get("slug").(string)
	if repoSlug == "" {
		repoSlug = d.Get("name").(string)
	}

	c := m.(Clients).genClient
	repoApi := c.ApiClient.RepositoriesApi

	res, err := repoApi.RepositoriesWorkspaceRepoSlugDelete(c.AuthContext, repoSlug, d.Get("owner").(string), nil)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// See https://confluence.atlassian.com/bbkb/what-is-a-repository-slug-1168845069.html
// Allows ASCII alphanumeric characters, underscores (_), en dashes (-), and periods (.) in repository slugs.
var slugForbiddenCharacters = regexp.MustCompile(`[^a-zA-Z0-9_.-]`)

// Limited to 62 characters.
var slugMaxCharacters = 62

// Can only end with an en dash if the entire repository slug is made up of en dashes (-).
var slugEnDashRightCharacters = regexp.MustCompile(`([^-])-+$`)

// A repository slug can also start with an en dash (-) if the entire repository slug is made up of en dashes.
var slugEnDashLeftCharacters = regexp.MustCompile(`^-+([^-])`)

// Does not allow consecutive en dashes (-) to be used in a repository slug unless the entire slug is made up of en dashes.
var slugEnDashConsecutive = regexp.MustCompile(`--+([^-])`)

func computeSlug(repoName string) string {
	slugTruncated := repoName
	if len(repoName) > slugMaxCharacters && strings.Contains(repoName, "-") {
		slugTruncated = repoName[:strings.LastIndex(repoName, "-")+1]
	}
	slugNormalized := slugForbiddenCharacters.ReplaceAllString(slugTruncated, "-")
	slugLeftDashNormalized := slugEnDashLeftCharacters.ReplaceAllString(slugNormalized, "$1")
	slugRightDashNormalized := slugEnDashRightCharacters.ReplaceAllString(slugLeftDashNormalized, "$1")
	slugEnDashConsecutiveNormalized := slugEnDashConsecutive.ReplaceAllString(slugRightDashNormalized, "-$1")
	return strings.ToLower(slugEnDashConsecutiveNormalized)
}

func splitFullName(repoFullName string) (string, string, error) {
	fullNameParts := strings.Split(repoFullName, "/")
	if len(fullNameParts) < 2 {
		return "", "", fmt.Errorf("Error parsing repo name (%s)", repoFullName)
	}
	owner := fullNameParts[0]
	repoSlug := strings.Join(fullNameParts[1:], "/")
	return owner, repoSlug, nil
}

func expandLinks(l []interface{}) *bitbucket.RepositoryLinks {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})

	if !ok {
		return nil
	}

	rp := &bitbucket.RepositoryLinks{}

	if v, ok := tfMap["avatar"].([]interface{}); ok && len(v) > 0 {
		rp.Avatar = expandLink(v)
	}

	return rp
}

func flattenLinks(rp *bitbucket.RepositoryLinks) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"avatar": flattenLink(rp.Avatar),
	}

	return []interface{}{m}
}

func expandLink(l []interface{}) *bitbucket.Link {

	tfMap, _ := l[0].(map[string]interface{})

	rp := &bitbucket.Link{}

	if v, ok := tfMap["href"].(string); ok {
		rp.Href = v
	}

	return rp
}

func flattenLink(rp *bitbucket.Link) []interface{} {
	m := map[string]interface{}{
		"href": rp.Href,
	}

	return []interface{}{m}
}

func createRepositoryInheritanceSettings(d *schema.ResourceData) *RepositoryInheritanceSettings {

	setting := &RepositoryInheritanceSettings{}

	// nolint:staticcheck
	if v, ok := d.GetOkExists("inherit_branching_model"); ok {
		model := v.(bool)
		setting.BranchingModel = &model
	}

	// nolint:staticcheck
	if v, ok := d.GetOkExists("inherit_default_merge_strategy"); ok {
		strategy := v.(bool)
		setting.DefaultMergeStrategy = &strategy
	}

	return setting
}

func repositoryId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE:REPO-SLUG", id)
	}

	return parts[0], parts[1], nil
}
