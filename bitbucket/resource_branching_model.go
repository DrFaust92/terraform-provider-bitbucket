package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// BranchingModel is the data we need to send to create a new branching model for the repository
type BranchingModel struct {
	Development *BranchModel `json:"development,omitempty"`
	Production  *BranchModel `json:"production,omitempty"`
}

type BranchModel struct {
	IsValid            bool   `json:"is_valid,omitempty"`
	Name               string `json:"name,omitempty"`
	UseMainbranch      bool   `json:"use_mainbranch,omitempty"`
	BranchDoesNotExist bool   `json:"branch_does_not_exist,omitempty"`
	Enabled            bool   `json:"enabled,omitempty"`
}

func resourceBranchingModel() *schema.Resource {
	return &schema.Resource{
		Create: resourceBranchingModelsPut,
		Read:   resourceBranchingModelsRead,
		Update: resourceBranchingModelsPut,
		Delete: resourceBranchingModelsDelete,

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
			"development": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"is_valid": {
							Type:     schema.TypeBool,
							Optional: true,
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
			// "production": {
			// 	Type:     schema.TypeList,
			// 	Optional: true,
			// 	MaxItems: 1,
			// 	Elem: &schema.Resource{
			// 		Schema: map[string]*schema.Schema{
			// 			"is_valid": {
			// 				Type:     schema.TypeBool,
			// 				Computed: true,
			// 			},
			// 			"name": {
			// 				Type:     schema.TypeString,
			// 				Optional: true,
			// 			},
			// 			"use_mainbranch": {
			// 				Type:     schema.TypeBool,
			// 				Optional: true,
			// 			},
			// 			"branch_does_not_exist": {
			// 				Type:     schema.TypeBool,
			// 				Optional: true,
			// 			},
			// 			"enabled": {
			// 				Type:     schema.TypeBool,
			// 				Optional: true,
			// 				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
			// 					oldBool, _ := strconv.ParseBool(old)
			// 					newBool, _ := strconv.ParseBool(new)

			// 					log.Printf("lol3: %s", old)
			// 					log.Printf("lol4: %s", new)

			// 					return !oldBool && newBool
			// 				},
			// 			},
			// 		},
			// 	},
			// },
		},
	}
}

func resourceBranchingModelsPut(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	branchingModel := expandBranchingModel(d)

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

	branchingModelsReq, _ := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/branching-model",
		d.Get("owner").(string),
		d.Get("repository").(string),
	))

	if branchingModelsReq.StatusCode == 200 {
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

		d.Set("development", flattenBranchModel(branchingModel.Development, "development"))
		// d.Set("production", flattenBranchModel(branchingModel.Production, "production"))
	}

	return nil
}

func resourceBranchingModelsDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	_, err := client.Put(fmt.Sprintf("2.0/repositories/%s/%s/branching-model/settings",
		d.Get("owner").(string),
		d.Get("repository").(string),
	), nil)

	if err != nil {
		return err
	}

	return err
}

func expandBranchingModel(d *schema.ResourceData) *BranchingModel {

	// users := make([]User, 0, len(d.Get("users").(*schema.Set).List()))

	// for _, item := range d.Get("users").(*schema.Set).List() {
	// 	users = append(users, User{Username: item.(string)})
	// }

	// groups := make([]Group, 0, len(d.Get("groups").(*schema.Set).List()))

	// for _, item := range d.Get("groups").(*schema.Set).List() {
	// 	m := item.(map[string]interface{})
	// 	groups = append(groups, Group{Owner: User{Username: m["owner"].(string)}, Slug: m["slug"].(string)})
	// }

	restict := &BranchingModel{}

	if v, ok := d.GetOk("development"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		restict.Development = expandBranchModel(v.([]interface{}))
	}

	if v, ok := d.GetOk("production"); ok && len(v.([]interface{})) > 0 && v.([]interface{}) != nil {
		restict.Production = expandBranchModel(v.([]interface{}))
	}

	return restict
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

	if typ == "production" {
		m["enabled"] = rp.Enabled
	}

	return []interface{}{m}
}
