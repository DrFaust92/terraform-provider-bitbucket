package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceProjectBranchingModel() *schema.Resource {
	return &schema.Resource{
		Create: resourceProjectBranchingModelsPut,
		Read:   resourceProjectBranchingModelsRead,
		Update: resourceProjectBranchingModelsPut,
		Delete: resourceProjectBranchingModelsDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"project": {
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

func resourceProjectBranchingModelsPut(d *schema.ResourceData, m interface{}) error {
	client := m.(Clients).httpClient
	branchingModel := expandBranchingModel(d)

	log.Printf("[DEBUG] Project Branching Model Request: %#v", branchingModel)
	bytedata, err := json.Marshal(branchingModel)

	if err != nil {
		return err
	}

	branchingModelReq, err := client.Put(fmt.Sprintf("2.0/workspaces/%s/projects/%s/branching-model/settings",
		d.Get("workspace").(string),
		d.Get("project").(string),
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

	d.SetId(string(fmt.Sprintf("%s/%s", d.Get("workspace").(string), d.Get("project").(string))))

	return resourceProjectBranchingModelsRead(d, m)
}

func resourceProjectBranchingModelsRead(d *schema.ResourceData, m interface{}) error {
	client := m.(Clients).httpClient

	workspace, repo, err := projectBranchingModelId(d.Id())
	if err != nil {
		return err
	}
	branchingModelsReq, _ := client.Get(fmt.Sprintf("2.0/workspaces/%s/projects/%s/branching-model", workspace, repo))

	if branchingModelsReq.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Project Branching Model (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if branchingModelsReq.Body == nil {
		return fmt.Errorf("error getting Project Branching Model (%s): empty response", d.Id())
	}

	var branchingModel *BranchingModel
	body, readerr := ioutil.ReadAll(branchingModelsReq.Body)
	if readerr != nil {
		return readerr
	}

	log.Printf("[DEBUG] Project Branching Model Response JSON: %v", string(body))

	decodeerr := json.Unmarshal(body, &branchingModel)
	if decodeerr != nil {
		return decodeerr
	}

	log.Printf("[DEBUG] Project Branching Model Response Decoded: %#v", branchingModel)

	d.Set("workspace", workspace)
	d.Set("project", repo)
	d.Set("development", flattenBranchModel(branchingModel.Development, "development"))
	d.Set("branch_type", flattenBranchTypes(branchingModel.BranchTypes))
	d.Set("production", flattenBranchModel(branchingModel.Production, "production"))

	return nil
}

func resourceProjectBranchingModelsDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(Clients).httpClient

	workspace, repo, err := projectBranchingModelId(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Put(fmt.Sprintf("2.0/workspaces/%s/projects/%s/branching-model/settings", workspace, repo), nil)

	if err != nil {
		return err
	}

	return err
}

func projectBranchingModelId(id string) (string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE/PROJECT", id)
	}

	return parts[0], parts[1], nil
}
