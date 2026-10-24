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

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// Deployment structure for handling key info
type Deployment struct {
	Name         string        `json:"name"`
	Stage        *Stage        `json:"environment_type"`
	UUID         string        `json:"uuid,omitempty"`
	Restrictions *Restrictions `json:"restrictions,omitempty"`
}

type Stage struct {
	Name string `json:"name"`
}

type Restrictions struct {
	AdminOnly bool `json:"admin_only"`
}

type Changes struct {
	Change *Change `json:"change"`
}

type Change struct {
	Name         string        `json:"name,omitempty"`
	Restrictions *Restrictions `json:"restrictions,omitempty"`
}

func resourceDeployment() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceDeploymentCreate,
		UpdateWithoutTimeout: resourceDeploymentUpdate,
		ReadWithoutTimeout:   resourceDeploymentRead,
		DeleteWithoutTimeout: resourceDeploymentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"stage": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"Test",
					"Staging",
					"Production",
				},
					false),
			},
			"repository": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(repositoryIdPattern, "must be WORKSPACE/REPO-SLUG, e.g. bitbucket_repository.example.id"),
			},
			"restrictions": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"admin_only": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func newDeploymentFromResource(d *schema.ResourceData) *Deployment {
	dk := &Deployment{
		Name: d.Get("name").(string),
		Stage: &Stage{
			Name: d.Get("stage").(string),
		},
	}

	if v, ok := d.GetOk("restrictions"); ok {
		rest := expandRestrictions(v.([]interface{}))
		dk.Restrictions = &rest
	}

	return dk
}

func resourceDeploymentCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	client := m.(Clients).httpClient
	rvcr := newDeploymentFromResource(d)
	bytedata, err := json.Marshal(rvcr)

	if err != nil {
		return diag.FromErr(err)
	}
	req, err := client.Post(fmt.Sprintf("2.0/repositories/%s/environments/",
		d.Get("repository").(string),
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return diag.FromErr(err)
	}

	var deployment Deployment

	body, readerr := io.ReadAll(req.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] deployment create res raw: %v", string(body))

	decodeerr := json.Unmarshal(body, &deployment)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] deployment create res decoded: %#v", deployment)

	d.Set("uuid", deployment.UUID)
	d.SetId(fmt.Sprintf("%s:%s", d.Get("repository"), deployment.UUID))

	return resourceDeploymentRead(ctx, d, m)
}

func resourceDeploymentRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {

	repoId, deployId, err := deploymentId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	client := m.(Clients).httpClient
	res, err := client.Get(fmt.Sprintf("2.0/repositories/%s/environments/%s",
		repoId,
		deployId,
	))

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Deployment (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	var deploy Deployment
	body, readerr := io.ReadAll(res.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] deployment response raw: %s", string(body))

	decodeerr := json.Unmarshal(body, &deploy)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	log.Printf("[DEBUG] deployment response decoded: %#v", deploy)

	d.Set("uuid", deploy.UUID)
	d.Set("name", deploy.Name)
	d.Set("stage", deploy.Stage.Name)
	d.Set("repository", repoId)
	d.Set("restrictions", flattenRestrictions(deploy.Restrictions))

	return nil
}

func resourceDeploymentUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	rvcr := &Changes{
		Change: &Change{},
	}

	if d.HasChange("name") {
		rvcr.Change.Name = d.Get("name").(string)
	}

	if d.HasChange("restrictions") {
		rest := expandRestrictions(d.Get("restrictions").([]interface{}))
		rvcr.Change.Restrictions = &rest
	}

	log.Printf("[DEBUG] deployment update req: %#v", rvcr)

	bytedata, err := json.Marshal(rvcr)

	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] deployment update req encoded: %v", string(bytedata))

	req, err := client.Post(fmt.Sprintf("2.0/repositories/%s/environments/%s/changes/",
		d.Get("repository").(string),
		d.Get("uuid").(string),
	), bytes.NewBuffer(bytedata))

	log.Printf("[DEBUG] deployment update res: %#v", req)

	if err != nil {
		return diag.FromErr(err)
	}

	if req.StatusCode >= 300 {
		return diag.Errorf("unexpected status updating deployment (%s): %s", d.Id(), req.Status)
	}

	return resourceDeploymentRead(ctx, d, m)
}

func resourceDeploymentDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	deployApi := c.ApiClient.DeploymentsApi

	workspaceId, repoId, err := deploymentRepoId(d.Get("repository").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := deployApi.DeleteEnvironmentForRepository(c.AuthContext, workspaceId, repoId, d.Get("uuid").(string))
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func expandRestrictions(conf []interface{}) Restrictions {
	tfMap, _ := conf[0].(map[string]interface{})

	target := Restrictions{
		AdminOnly: tfMap["admin_only"].(bool),
	}

	return target
}

func flattenRestrictions(rp *Restrictions) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"admin_only": rp.AdminOnly,
	}

	return []interface{}{m}
}

var (
	repositoryIdPattern = regexp.MustCompile(`^[^/]+/[^/:]+$`)
	deploymentIdPattern = regexp.MustCompile(`^[^/]+/[^/:]+:[^/:]+$`)
)

func deploymentId(id string) (string, string, error) {
	parts := strings.Split(id, ":")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE/REPO-SLUG:DEPLOYMENT-UUID", id)
	}

	return parts[0], parts[1], nil
}

func deploymentRepoId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE-ID/REPO-ID", id)
	}

	return parts[0], parts[1], nil
}
