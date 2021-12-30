package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// BranchingModel is the data we need to send to create a new branching model for the repository
type BranchingModel struct {
	Development *BranchModel  `json:"development,omitempty"`
	Production  *BranchModel  `json:"production,omitempty"`
	BranchTypes []*BranchType `json:"branch_types"`
}

type BranchModel struct {
	IsValid            bool   `json:"is_valid,omitempty"`
	Name               string `json:"name,omitempty"`
	UseMainbranch      bool   `json:"use_mainbranch,omitempty"`
	BranchDoesNotExist bool   `json:"branch_does_not_exist,omitempty"`
	Enabled            bool   `json:"enabled,omitempty"`
}

type BranchType struct {
	Enabled bool   `json:"enabled,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
}

func resourceBranchingModel() *schema.Resource {
	return &schema.Resource{
		Create: resourceBranchingModelsPut,
		Read:   resourceBranchingModelsRead,
		Update: resourceBranchingModelsPut,
		Delete: resourceBranchingModelsDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
		},
	}
}

func resourceBranchingModelsPut(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	branchingModel := expandBranchingModel(d)

	log.Printf("[DEBUG] Branching Model Request: %#v", branchingModel)
	bytedata, err := json.Marshal(branchingModel)

	if err != nil {
		return err
	}

	branchingModelReq, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s/branching-model/settings",
		d.Get("owner").(string),
		d.Get("repository").(string),
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return err
	}

	body, readerr := ioutil.ReadAll(branchingModelReq.Body)
	if readerr != nil {
		return readerr
	}

	decodeerr := json.Unmarshal(body, &branchingModel)
	if decodeerr != nil {
		return decodeerr
	}

	d.SetId(string(fmt.Sprintf("%s/%s", d.Get("owner").(string), d.Get("repository").(string))))

	return resourceBranchingModelsRead(d, m)
}

func resourceBranchingModelsRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	owner, repo, err := branchingModelId(d.Id())
	if err != nil {
		return err
	}
	branchingModelsReq, _ := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/branching-model", owner, repo))

	if branchingModelsReq.StatusCode == 404 {
		log.Printf("[WARN] Branching Model (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if branchingModelsReq.Body == nil {
		return fmt.Errorf("error getting Branching Model (%s): empty response", d.Id())
	}

	var branchingModel *BranchingModel
	body, readerr := ioutil.ReadAll(branchingModelsReq.Body)
	if readerr != nil {
		return readerr
	}

	log.Printf("[DEBUG] Branching Model Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &branchingModel)
	if decodeerr != nil {
		return decodeerr
	}

	log.Printf("[DEBUG] Branching Model Response Decoded: %#v", branchingModel)

	d.Set("owner", owner)
	d.Set("repository", repo)
	d.Set("development", flattenBranchModel(branchingModel.Development, "development"))
	d.Set("branch_type", flattenBranchTypes(branchingModel.BranchTypes))
	d.Set("production", flattenBranchModel(branchingModel.Production, "production"))

	return nil
}

func resourceBranchingModelsDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)

	owner, repo, err := branchingModelId(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Put(fmt.Sprintf("2.0/repositories/%s/%s/branching-model/settings", owner, repo), nil)

	if err != nil {
		return err
	}

	return err
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
		rp.Name = v
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

	// if production branch is disabled it wont show up in response and will show up without the proerty if enabled
	if typ == "production" {
		m["enabled"] = true
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
			bt.Enabled = v
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
			"enabled": true,
		}

		tfList = append(tfList, branchType)
	}

	return tfList
}

func branchingModelId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected format of ID (%q), expected OWNER/REPOSITORY", id)
	}

	return parts[0], parts[1], nil
}
