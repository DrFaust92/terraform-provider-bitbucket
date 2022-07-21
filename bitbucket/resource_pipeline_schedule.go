package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/migara/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePipelineSchedule() *schema.Resource {
	return &schema.Resource{
		Create: resourcePipelineScheduleCreate,
		Read:   resourcePipelineScheduleRead,
		Update: resourcePipelineScheduleUpdate,
		Delete: resourcePipelineScheduleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"cron_pattern": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ref_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"ref_type": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{"branch", "tag"}, false),
						},
						"selector": {
							Type:     schema.TypeList,
							Required: true,
							ForceNew: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"pattern": {
										Type:     schema.TypeString,
										Required: true,
										ForceNew: true,
									},
								},
							},
						},
					},
				},
			},
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePipelineScheduleCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	pipeSchedule := expandPipelineSchedule(d)
	log.Printf("[DEBUG] Pipeline Schedule Request: %#v", pipeSchedule)

	repo := d.Get("repository").(string)
	workspace := d.Get("workspace").(string)
	schedule, _, err := pipeApi.CreateRepositoryPipelineSchedule(c.AuthContext, *pipeSchedule, workspace, repo)

	if err != nil {
		return fmt.Errorf("error creating pipeline schedule: %w", err)
	}

	d.SetId(string(fmt.Sprintf("%s/%s/%s", workspace, repo, schedule.Uuid)))

	return resourcePipelineScheduleRead(d, m)
}

func resourcePipelineScheduleUpdate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, uuid, err := pipeScheduleId(d.Id())
	if err != nil {
		return err
	}

	pipeSchedule := expandPipelineSchedule(d)
	log.Printf("[DEBUG] Pipeline Schedule Request: %#v", pipeSchedule)
	_, _, err = pipeApi.UpdateRepositoryPipelineSchedule(c.AuthContext, *pipeSchedule, workspace, repo, uuid)

	if err != nil {
		return fmt.Errorf("error updating pipeline schedule: %w", err)
	}

	return resourcePipelineScheduleRead(d, m)
}

func resourcePipelineScheduleRead(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, uuid, err := pipeScheduleId(d.Id())
	if err != nil {
		return err
	}

	schedule, res, err := pipeApi.GetRepositoryPipelineSchedule(c.AuthContext, workspace, repo, uuid)
	if err != nil {
		return fmt.Errorf("error reading Pipeline Schedule (%s): %w", d.Id(), err)
	}

	if res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Pipeline Schedule (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if res.Body == nil {
		return fmt.Errorf("error getting Pipeline Schedule (%s): empty response", d.Id())
	}

	d.Set("repository", repo)
	d.Set("workspace", workspace)
	d.Set("uuid", schedule.Uuid)
	d.Set("enabled", schedule.Enabled)
	d.Set("cron_pattern", schedule.CronPattern)

	d.Set("target", flattenPipelineRefTarget(schedule.Target))

	return nil
}

func resourcePipelineScheduleDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	workspace, repo, uuid, err := pipeScheduleId(d.Id())
	if err != nil {
		return err
	}
	_, err = pipeApi.DeleteRepositoryPipelineSchedule(c.AuthContext, workspace, repo, uuid)

	if err != nil {
		return fmt.Errorf("error deleting Pipeline Schedule (%s): %w", d.Id(), err)
	}

	return err
}

func expandPipelineSchedule(d *schema.ResourceData) *bitbucket.PipelineSchedule {
	schedule := &bitbucket.PipelineSchedule{
		Enabled:     d.Get("enabled").(bool),
		CronPattern: d.Get("cron_pattern").(string),
		Target:      expandPipelineRefTarget(d.Get("target").([]interface{})),
	}

	return schedule
}

func expandPipelineRefTarget(conf []interface{}) *bitbucket.PipelineRefTarget {
	tfMap, _ := conf[0].(map[string]interface{})

	target := &bitbucket.PipelineRefTarget{
		RefName:  tfMap["ref_name"].(string),
		RefType:  tfMap["ref_type"].(string),
		Selector: expandPipelineRefTargetSelector(tfMap["selector"].([]interface{})),
		Type_:    "pipeline_ref_target",
	}

	return target
}

func expandPipelineRefTargetSelector(conf []interface{}) *bitbucket.PipelineSelector {
	tfMap, _ := conf[0].(map[string]interface{})

	selector := &bitbucket.PipelineSelector{
		Pattern: tfMap["pattern"].(string),
		Type_:   "branches",
	}

	return selector
}

func flattenPipelineRefTarget(rp *bitbucket.PipelineRefTarget) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"ref_name": rp.RefName,
		"ref_type": rp.RefType,
		"selector": flattenPipelineSelector(rp.Selector),
	}

	return []interface{}{m}
}

func flattenPipelineSelector(rp *bitbucket.PipelineSelector) []interface{} {
	if rp == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"pattern": rp.Pattern,
	}

	return []interface{}{m}
}

func pipeScheduleId(id string) (string, string, string, error) {
	parts := strings.Split(id, "/")

	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected format of ID (%q), expected WORKSPACE-ID/REPO-ID/UUID", id)
	}

	return parts[0], parts[1], parts[2], nil
}
