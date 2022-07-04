package bitbucket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
	"time"
)

// DeploymentVariable structure for handling key info
type DeploymentVariable struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	UUID    string `json:"uuid,omitempty"`
	Secured bool   `json:"secured"`
}

// PaginatedReviewers is a paginated list that the bitbucket api returns
type PaginatedDeploymentVariables struct {
	Values []DeploymentVariable `json:"values,omitempty"`
	Page   int                  `json:"page,omitempty"`
	Size   int                  `json:"size,omitempty"`
	Next   string               `json:"next,omitempty"`
}

func resourceDeploymentVariable() *schema.Resource {
	return &schema.Resource{
		Create: resourceDeploymentVariableCreate,
		Update: resourceDeploymentVariableUpdate,
		Read:   resourceDeploymentVariableRead,
		Delete: resourceDeploymentVariableDelete,

		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key": {
				Type:     schema.TypeString,
				Required: true,
			},
			"value": {
				Type:     schema.TypeString,
				Required: true,
			},
			"secured": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"deployment": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func newDeploymentVariableFromResource(d *schema.ResourceData) *DeploymentVariable {
	dk := &DeploymentVariable{
		Key:     d.Get("key").(string),
		Value:   d.Get("value").(string),
		Secured: d.Get("secured").(bool),
	}
	return dk
}

func parseDeploymentId(str string) (repository string, deployment string) {
	parts := strings.SplitN(str, ":", 2)
	return parts[0], parts[1]
}

func resourceDeploymentVariableCreate(d *schema.ResourceData, m interface{}) error {

	client := m.(*Client)
	rvcr := newDeploymentVariableFromResource(d)
	bytedata, err := json.Marshal(rvcr)

	if err != nil {
		return err
	}
	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	req, err := client.Post(fmt.Sprintf("2.0/repositories/%s/deployments_config/environments/%s/variables",
		repository,
		deployment,
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return err
	}

	var rv DeploymentVariable

	body, readerr := ioutil.ReadAll(req.Body)
	if readerr != nil {
		return readerr
	}

	decodeerr := json.Unmarshal(body, &rv)
	if decodeerr != nil {
		return decodeerr
	}
	d.Set("uuid", rv.UUID)
	d.SetId(rv.UUID)

	time.Sleep(5000 * time.Millisecond) // sleep for a while, to allow BitBucket cache to catch up
	return resourceDeploymentVariableRead(d, m)
}

func resourceDeploymentVariableRead(d *schema.ResourceData, m interface{}) error {

	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	client := m.(*Client)
	// @TODO: Handle pagination instead of hard-code current variable limit
	rvReq, _ := client.Get(fmt.Sprintf("2.0/repositories/%s/deployments_config/environments/%s/variables?pagelen=50",
		repository,
		deployment,
	))

	log.Printf("ID: %s", url.PathEscape(d.Id()))

	if rvReq.StatusCode == 200 {
		var prv PaginatedDeploymentVariables
		body, readerr := ioutil.ReadAll(rvReq.Body)
		if readerr != nil {
			return readerr
		}

		decodeerr := json.Unmarshal(body, &prv)
		if decodeerr != nil {
			return decodeerr
		}

		if prv.Size < 1 {
			d.SetId("")
			return nil
		}

		var uuid = d.Get("uuid").(string)
		for _, rv := range prv.Values {
			if rv.UUID == uuid {
				d.SetId(rv.UUID)
				d.Set("key", rv.Key)
				d.Set("value", rv.Value)
				d.Set("secured", rv.Secured)
				return nil
			}
		}
		d.SetId("")
	}

	if rvReq.StatusCode == 404 {
		d.SetId("")
		return nil
	}

	return nil
}

func resourceDeploymentVariableUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	rvcr := newDeploymentVariableFromResource(d)
	bytedata, err := json.Marshal(rvcr)

	if err != nil {
		return err
	}
	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	req, err := client.Put(fmt.Sprintf("2.0/repositories/%s/deployments_config/environments/%s/variables/%s",
		repository,
		deployment,
		d.Get("uuid").(string),
	), bytes.NewBuffer(bytedata))

	if err != nil {
		return err
	}

	if req.StatusCode != 200 {
		return nil
	}

	return resourceDeploymentVariableRead(d, m)
}

func resourceDeploymentVariableDelete(d *schema.ResourceData, m interface{}) error {
	repository, deployment := parseDeploymentId(d.Get("deployment").(string))
	client := m.(*Client)
	_, err := client.Delete(fmt.Sprintf(fmt.Sprintf("2.0/repositories/%s/deployments_config/environments/%s/variables/%s",
		repository,
		deployment,
		d.Get("uuid").(string),
	)))
	return err
}
