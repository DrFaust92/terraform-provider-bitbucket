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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// BranchingModel is the data we need to send to create a new branching model for the repository
type BranchingModel struct {
	Development           *BranchModel  `json:"development,omitempty"`
	Production            *BranchModel  `json:"production,omitempty"`
	BranchTypes           []*BranchType `json:"branch_types"`
	DefaultBranchDeletion *FlexBool     `json:"default_branch_deletion,omitempty"`
}

type BranchModel struct {
	IsValid            bool    `json:"is_valid,omitempty"`
	Name               *string `json:"name"`
	UseMainbranch      bool    `json:"use_mainbranch"`
	BranchDoesNotExist bool    `json:"branch_does_not_exist,omitempty"`
	Enabled            bool    `json:"enabled,omitempty"`
}

type BranchType struct {
	// TRICKY: This is a pointer to a bool because the API omits the field if
	// it is true. json.Unmarshal treats a missing field as false, so we need to
	// handle this case explicitly.
	Enabled *bool  `json:"enabled"`
	Kind    string `json:"kind,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
}

func resourceBranchingModel() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceBranchingModelsPut,
		ReadWithoutTimeout:   resourceBranchingModelsRead,
		UpdateWithoutTimeout: resourceBranchingModelsPut,
		DeleteWithoutTimeout: resourceBranchingModelsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
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
			"branch_type": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 4,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"kind": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"feature", "bugfix", "release", "hotfix"}, false),
						},
						"prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"development": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"is_valid": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"use_mainbranch": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"branch_does_not_exist": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"production": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"is_valid": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"use_mainbranch": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"branch_does_not_exist": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
			},
			"default_branch_deletion": {
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceBranchingModelsPut(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient
	branchingModel := expandBranchingModel(d)

	log.Printf("[DEBUG] Branching Model Request: %#v", branchingModel)
	bytedata, err := json.Marshal(branchingModel)

	if err != nil {
		return diag.FromErr(err)
	}

	branchingModelReq, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s/branching-model/settings",
		d.Get("owner").(string),
		d.Get("repository").(string),
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return diag.FromErr(err)
	}

	body, readerr := io.ReadAll(branchingModelReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	decodeerr := json.Unmarshal(body, &branchingModel)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	d.SetId(fmt.Sprintf("%s/%s", d.Get("owner").(string), d.Get("repository").(string)))

	return resourceBranchingModelsRead(ctx, d, m)
}

func resourceBranchingModelsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	owner, repo, err := branchingModelId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	branchingModelsReq, _ := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/branching-model/settings", owner, repo))

	if branchingModelsReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Branching Model (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if branchingModelsReq.Body == nil {
		return diag.Errorf("error getting Branching Model (%s): empty response", d.Id())
	}

	var branchingModel *BranchingModel
	body, readerr := io.ReadAll(branchingModelsReq.Body)
	if readerr != nil {
		return diag.FromErr(readerr)
	}

	log.Printf("[DEBUG] Branching Model Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &branchingModel)
	if decodeerr != nil {
		return diag.FromErr(decodeerr)
	}

	// Set default value for Enabled if it is nil
	for _, branchType := range branchingModel.BranchTypes {
		if branchType.Enabled == nil {
			defaultTrue := true
			branchType.Enabled = &defaultTrue
		}
	}

	log.Printf("[DEBUG] Branching Model Response Decoded: %#v", branchingModel)

	d.Set("owner", owner)
	d.Set("repository", repo)
	d.Set("default_branch_deletion", branchingModel.DefaultBranchDeletion.BoolPtr())
	d.Set("development", flattenBranchModel(branchingModel.Development, "development"))
	d.Set("branch_type", flattenBranchTypes(branchingModel.BranchTypes))
	d.Set("production", flattenBranchModel(branchingModel.Production, "production"))

	return nil
}

func resourceBranchingModelsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(Clients).httpClient

	owner, repo, err := branchingModelId(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/branching-model/settings", owner, repo), nil)

	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(err)
}

func expandBranchingModel(d *schema.ResourceData) *BranchingModel {
	model := &BranchingModel{}

	if v, ok := d.GetOk("development"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		model.Development = expandBranchModel(v.([]interface{}))
	}

	if v, ok := d.GetOk("production"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		model.Production = expandBranchModel(v.([]interface{}))
	}

	if v, ok := d.GetOk("branch_type"); ok && v.(*schema.Set).Len() > 0 {
		model.BranchTypes = expandBranchTypes(v.(*schema.Set))
	} else {
		model.BranchTypes = make([]*BranchType, 0)
	}

	//nolint:staticcheck
	if v, ok := d.GetOkExists("default_branch_deletion"); ok {
		val := v.(bool)
		model.DefaultBranchDeletion = &FlexBool{Value: &val}
	}

	return model
}

func expandBranchModel(l []interface{}) *BranchModel {
	if len(l) == 0 || l[0] == nil {
		return nil
	}

	tfMap, ok := l[0].(map[string]interface{})

	if !ok {
		return nil
	}

	rp := &BranchModel{}

	if v, ok := tfMap["name"].(string); ok {
		if v == "" {
			rp.Name = nil
		} else {
			rp.Name = &v
		}
	}

	if v, ok := tfMap["enabled"].(bool); ok {
		rp.Enabled = v
	}

	if v, ok := tfMap["branch_does_not_exist"].(bool); ok {
		rp.BranchDoesNotExist = v
	}

	if v, ok := tfMap["use_mainbranch"].(bool); ok {
		rp.UseMainbranch = v
	}

	return rp
}

func flattenBranchModel(rp *BranchModel, typ string) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"branch_does_not_exist": rp.BranchDoesNotExist,
		"is_valid":              rp.IsValid,
		"use_mainbranch":        rp.UseMainbranch,
		"name":                  rp.Name,
	}

	// Production has an "enabled" field that is not present in development.
	if typ == "production" {
		m["enabled"] = rp.Enabled
	}

	return []interface{}{m}
}

func expandBranchTypes(tfList *schema.Set) []*BranchType {
	if tfList.Len() == 0 {
		return nil
	}

	var branchTypes []*BranchType

	for _, tfMapRaw := range tfList.List() {
		tfMap, ok := tfMapRaw.(map[string]interface{})

		if !ok {
			continue
		}

		bt := &BranchType{
			Kind: tfMap["kind"].(string),
		}

		if v, ok := tfMap["prefix"].(string); ok {
			bt.Prefix = v
		}

		if v, ok := tfMap["enabled"].(bool); ok {
			bt.Enabled = &v
		}

		branchTypes = append(branchTypes, bt)
	}

	return branchTypes
}

func flattenBranchTypes(branchTypes []*BranchType) []interface{} {
	if len(branchTypes) == 0 {
		return nil
	}

	var tfList []interface{}

	for _, btRaw := range branchTypes {
		log.Printf("[DEBUG] Branch Type Response Decoded: %#v", btRaw)

		if btRaw == nil {
			continue
		}

		branchType := map[string]interface{}{
			"kind":    btRaw.Kind,
			"prefix":  btRaw.Prefix,
			"enabled": btRaw.Enabled,
		}

		tfList = append(tfList, branchType)
	}

	return tfList
}

func branchingModelId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected OWNER/REPOSITORY", id)
	}

	return parts[0], parts[1], nil
}
