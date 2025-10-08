package bitbucket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataFile() *schema.Resource {
	return &schema.Resource{
		ReadWithoutTimeout: dataFileRead,
		Description:        "Datasource to retrieve file content or metadata information",
		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:         schema.TypeString,
				Description:  "Workspace slug or UUID",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"repo_slug": {
				Type:         schema.TypeString,
				Description:  "Repo slug or UUID",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"commit": {
				Type:        schema.TypeString,
				Description: "Commit hash or branch name",
				Required:    true,
			},
			"path": {
				Type:        schema.TypeString,
				Description: "Path to file (starting from commit)",
				Required:    true,
			},
			"format": {
				Type:         schema.TypeString,
				Description:  "Format if file to return: content/base64 content or metadata.",
				Optional:     true,
				Default:      "raw",
				ValidateFunc: validation.StringInSlice([]string{"meta", "raw"}, false),
			},
			"include_links": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to include the links for the file metadata or not.",
			},
			"include_commit": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to include the commit for the file metadata or not.",
			},
			"include_commit_links": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to include the commit links for the file metadata or not.",
			},
			"content": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Raw string content of path return (not escaped).",
			},
			"content_b64": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Base64-encoded version of path return, safe for embedding.",
			},
			"metadata": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Parsed metadata of path (JSON/XML), if available",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"path": {
							Type:        schema.TypeString,
							Description: "Path to object",
							Computed:    true,
						},
						"type": {
							Type:        schema.TypeString,
							Description: "Type of object",
							Computed:    true,
						},
						"escaped_path": {
							Type:        schema.TypeString,
							Description: "Escaped path of object",
							Optional:    true,
							Computed:    true,
						},
						"mime_type": {
							Type:        schema.TypeString,
							Description: "Mimetype",
							Optional:    true,
							Computed:    true,
						},
						"size": {
							Type:        schema.TypeInt,
							Description: "Size of object",
							Optional:    true,
							Computed:    true,
						},
						"commit": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "Commit information for path",
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Description: "Type of commit",
										Computed:    true,
									},
									"hash": {
										Type:        schema.TypeString,
										Description: "Hash for commit",
										Optional:    true,
										Computed:    true,
									},
									"link": {
										Type:        schema.TypeList,
										Optional:    true,
										Computed:    true,
										Description: "Link information for commit",
										MaxItems:    2,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"self": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"href": {
																Type:        schema.TypeString,
																Description: "URL to commit link",
																Computed:    true,
															},
														},
													},
												},
												"html": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"href": {
																Type:        schema.TypeString,
																Description: "URL to commit link",
																Computed:    true,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"link": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Description: "Link information to path",
							MaxItems:    3,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"self": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"href": {
													Type:        schema.TypeString,
													Description: "URL to link",
													Computed:    true,
												},
											},
										},
									},
									"meta": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"href": {
													Type:        schema.TypeString,
													Description: "URL to link",
													Computed:    true,
												},
											},
										},
									},
									"history": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"href": {
													Type:        schema.TypeString,
													Description: "URL to link",
													Computed:    true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

type FileValue struct {
	Path        optional.String `json:"path,omitempty"`
	Type        string          `json:"type"`
	Commit      *CommitValue    `json:"commit,omitempty"`
	Size        optional.Int64  `json:"size,omitempty"`
	EscapedPath optional.String `json:"escaped_path,omitempty"`
	MimeType    optional.String `json:"mimetype,omitempty"`
	Links       *LinksValue     `json:"links,omitempty"`
}

type CommitValue struct {
	Type  string            `json:"type"`
	Hash  optional.String   `json:"hash,omitempty"`
	Links *CommitLinksValue `json:"links,omitempty"`
}

type CommitLinksValue struct {
	Self *FileHref `json:"self"`
	Html *FileHref `json:"html,omitempty"`
}

type LinksValue struct {
	Self    *FileHref `json:"self"`
	Meta    *FileHref `json:"meta,omitempty"`
	History *FileHref `json:"history,omitempty"`
}

type FileHref struct {
	Href string `json:"href"`
}

func dataFileRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	repoSlug := d.Get("repo_slug").(string)
	workspace := d.Get("workspace").(string)
	commit := d.Get("commit").(string)
	path := d.Get("path").(string)
	format := d.Get("format").(string)
	include_links := d.Get("include_links").(bool)
	include_commit := d.Get("include_commit").(bool)
	include_commit_links := d.Get("include_commit_links").(bool)

	if include_commit_links && !include_commit {
		return diag.Errorf("include_commit_links cannot be true if include_commit is not set to true.")
	}

	url := fmt.Sprintf("2.0/repositories/%s/%s/src/%s/%s",
		workspace,
		repoSlug,
		commit,
		path,
	)

	if format == "meta" {
		url += "?format=meta"
	}

	res, err := client.Get(url)
	if err != nil {
		return diag.FromErr(err)
	}

	if res == nil {
		return diag.Errorf("no response returned from repositories src commit call. Make sure your credentials are accurate.")
	}

	if res.StatusCode == http.StatusNotFound {
		return diag.Errorf("unable to locate file with params (%s): ", dumpResourceData(d, dataFile().Schema))
	}

	if res.Body == nil {
		return diag.Errorf("error reading file information with params (%s): ", dumpResourceData(d, dataFile().Schema))
	}

	fileBody, readerr := io.ReadAll(res.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	if format != "meta" {
		d.Set("content", string(fileBody))
		d.Set("content_b64", base64.StdEncoding.EncodeToString(fileBody))
	} else {
		metadata, err := processJson(fileBody, res.Header.Get("Content-Type"), include_commit, include_commit_links, include_links)
		if err != nil {
			return diag.FromErr(err)
		}

		d.Set("metadata", []interface{}{metadata})
	}
	d.SetId(fmt.Sprintf("%s/%s", commit, path))
	return nil
}

// Processes the metadata JSON
func processJson(fileBody []byte, contentType string, include_commit bool, include_commit_links bool, include_links bool) (interface{}, error) {

	if fileBody == nil {
		return nil, fmt.Errorf("missing response body")
	}
	var fileValue FileValue
	err := fileValue.decode(fileBody, contentType)
	if err != nil {
		return nil, fmt.Errorf("error while attempting to decode file metadata: %v", err)
	}
	return flattenFileReturn(&fileValue, include_commit, include_commit_links, include_links), nil
}

// Flattens the file return
func flattenFileReturn(file *FileValue, include_commit bool, include_commit_links bool, include_links bool) map[string]interface{} {
	if file == nil {
		return nil
	}

	metadata := map[string]interface{}{
		"path":         file.Path.Default(""),
		"type":         file.Type,
		"size":         file.Size.Default(0),
		"escaped_path": file.EscapedPath.Default(""),
		"mime_type":    file.MimeType.Default(""),
	}

	if file.Links != nil && include_links {
		links := map[string]interface{}{}
		links["self"] = []interface{}{
			map[string]interface{}{
				"href": file.Links.Self.Href,
			},
		}
		if file.Links.Meta != nil {
			links["meta"] = []interface{}{
				map[string]interface{}{
					"href": file.Links.Meta.Href,
				},
			}
		}
		if file.Links.History != nil {
			links["history"] = []interface{}{
				map[string]interface{}{
					"href": file.Links.History.Href,
				},
			}
		}
		metadata["link"] = []interface{}{links}
	}

	if file.Commit != nil && include_commit {
		commit := map[string]interface{}{
			"type": file.Commit.Type,
			"hash": file.Commit.Hash.Default(""),
		}
		if file.Commit.Links != nil && include_commit_links {
			links := map[string]interface{}{}
			links["self"] = []interface{}{
				map[string]interface{}{
					"href": file.Commit.Links.Self.Href,
				},
			}
			if file.Commit.Links.Html != nil {
				links["html"] = []interface{}{
					map[string]interface{}{
						"href": file.Commit.Links.Html.Href,
					},
				}
			}
			commit["link"] = []interface{}{links}
		}
		metadata["commit"] = []interface{}{commit}
	}
	return metadata
}

// Utility to extract a flat map of all schema values
func dumpResourceData(d *schema.ResourceData, schema map[string]*schema.Schema) string {
	data := map[string]interface{}{}

	for k := range schema {
		if v, ok := d.GetOk(k); ok {
			data[k] = v
		}
	}

	bytes, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Sprintf("failed to marshal resource data: %v", err)
	}
	return string(bytes)
}

// Create custom unmarshaller so it supports optional struct values
func (f *FileValue) decode(data []byte, contentType string) (err error) {
	switch {
	case strings.Contains(contentType, "json"):
		type commitAlias struct {
			Type  string            `json:"type"`
			Hash  *string           `json:"hash,omitempty"`
			Links *CommitLinksValue `json:"links,omitempty"`
		}

		type fileAlias struct {
			Path        *string      `json:"path,omitempty"`
			Type        string       `json:"type"`
			Commit      *commitAlias `json:"commit,omitempty"`
			Size        *int64       `json:"size,omitempty"`
			EscapedPath *string      `json:"escaped_path,omitempty"`
			MimeType    *string      `json:"mimetype,omitempty"`
			Links       *LinksValue  `json:"links,omitempty"`
		}

		var aux fileAlias

		if err := json.Unmarshal(data, &aux); err != nil {
			return err
		}

		// Convert basic fields
		f.Type = aux.Type
		f.Path = toOptionalString(aux.Path)
		f.Size = toOptionalInt64(aux.Size)
		f.EscapedPath = toOptionalString(aux.EscapedPath)
		f.MimeType = toOptionalString(aux.MimeType)
		f.Links = aux.Links

		// Convert nested commit
		if aux.Commit != nil {
			f.Commit = &CommitValue{
				Type:  aux.Commit.Type,
				Hash:  toOptionalString(aux.Commit.Hash),
				Links: aux.Commit.Links,
			}
		}
	default:
		return fmt.Errorf("unknown metadata content type %s", contentType)
	}
	return nil
}

// Convert pointer to optional string
func toOptionalString(p *string) optional.String {
	if p == nil {
		return optional.EmptyString()
	}

	return optional.NewString(*p)
}

// Convert pointer to optional int
func toOptionalInt64(i *int64) optional.Int64 {
	if i == nil {
		return optional.EmptyInt64()
	}

	return optional.NewInt64(*i)
}
