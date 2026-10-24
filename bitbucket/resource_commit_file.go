package bitbucket

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceCommitFile() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceCommitFilePut,
		ReadWithoutTimeout:   resourceCommitFileRead,
		DeleteWithoutTimeout: resourceCommitFileDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repo_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"content": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"filename": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"branch": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"commit_message": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The SHA of the commit that modified the file",
				ForceNew:    true,
			},
			"commit_author": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The SHA of the commit that modified the file",
				ForceNew:    true,
			},
			"commit_sha": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The SHA of the commit that modified the file",
			},
		},
	}
}

func resourceCommitFilePut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	repoSlug := d.Get("repo_slug").(string)
	workspace := d.Get("workspace").(string)
	content := d.Get("content").(string)
	filename := d.Get("filename").(string)
	branch := d.Get("branch").(string)
	commitMessage := d.Get("commit_message").(string)
	commitAuthor := d.Get("commit_author").(string)

	unlock := lockCommitBranch(workspace, repoSlug, branch)
	defer unlock()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile(filename, filename)
	_, err := part.Write([]byte(content))
	if err != nil {
		return diag.FromErr(err)
	}

	messageFormField, err := writer.CreateFormField("message")
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = messageFormField.Write([]byte(commitMessage))
	if err != nil {
		return diag.FromErr(err)
	}
	authorFormField, err := writer.CreateFormField("author")
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = authorFormField.Write([]byte(commitAuthor))
	if err != nil {
		return diag.FromErr(err)
	}

	branchFormField, err := writer.CreateFormField("branch")
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = branchFormField.Write([]byte(branch))
	if err != nil {
		return diag.FromErr(err)
	}
	if err := writer.Close(); err != nil {
		return diag.FromErr(err)
	}

	res, err := client.PostWithContentType(fmt.Sprintf("2.0/repositories/%s/%s/src",
		workspace,
		repoSlug,
	), writer.FormDataContentType(), body)

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	if res.StatusCode != http.StatusCreated {
		return diag.Errorf("unexpected status committing %s to %s/%s@%s: %s", filename, workspace, repoSlug, branch, res.Status)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s/%s", workspace, repoSlug, branch, filename))

	location, err := res.Location()
	if err != nil {
		return diag.Errorf("committing %s to %s/%s@%s: no commit location in response: %s", filename, workspace, repoSlug, branch, err)
	}
	splitPath := strings.Split(location.Path, "/")
	d.Set("commit_sha", splitPath[len(splitPath)-1])

	return resourceCommitFileRead(ctx, d, m)
}

func resourceCommitFileRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	sourceApi := c.ApiClient.SourceApi

	repoSlug := d.Get("repo_slug").(string)
	workspace := d.Get("workspace").(string)
	filename := d.Get("filename").(string)
	commit := d.Get("commit_sha").(string)

	_, res, err := sourceApi.RepositoriesWorkspaceRepoSlugSrcCommitPathGet(c.AuthContext, commit, filename, repoSlug, workspace, &bitbucket.SourceApiRepositoriesWorkspaceRepoSlugSrcCommitPathGetOpts{})

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceCommitFileDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

var commitBranchLocks sync.Map

func lockCommitBranch(workspace, repoSlug, branch string) func() {
	mu, _ := commitBranchLocks.LoadOrStore(workspace+"/"+repoSlug+"@"+branch, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	return mu.(*sync.Mutex).Unlock
}
