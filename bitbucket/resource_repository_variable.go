package bitbucket

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/migara/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRepositoryVariable() *schema.Resource {
	return &schema.Resource{
		Create: resourceRepositoryVariableCreate,
		Update: resourceRepositoryVariableUpdate,
		Read:   resourceRepositoryVariableRead,
		Delete: resourceRepositoryVariableDelete,

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
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"secured": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func newRepositoryVariableFromResource(d *schema.ResourceData) bitbucket.PipelineVariable {
	dk := bitbucket.PipelineVariable{
		Key:     d.Get("key").(string),
		Value:   d.Get("value").(string),
		Secured: d.Get("secured").(bool),
	}
	return dk
}

func resourceRepositoryVariableCreate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi
	rvcr := newRepositoryVariableFromResource(d)

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return err
	}

	rvRes, _, err := pipeApi.CreateRepositoryPipelineVariable(c.AuthContext, rvcr, workspace, repoSlug)

	if err != nil {
		return fmt.Errorf("error creating Repository Variable (%s): %w", repo, err)
	}

	d.Set("uuid", rvRes.Uuid)
	d.SetId(rvRes.Key)

	return resourceRepositoryVariableRead(d, m)
}

func resourceRepositoryVariableRead(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return err
	}

	rvRes, res, err := pipeApi.GetRepositoryPipelineVariable(c.AuthContext, workspace, repoSlug, d.Get("uuid").(string))
	if err != nil {
		return fmt.Errorf("error reading Repository Variable (%s): %w", d.Id(), err)
	}
	if res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Repository Variable (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("uuid", rvRes.Uuid)
	d.Set("key", rvRes.Key)
	d.Set("secured", rvRes.Secured)

	if !rvRes.Secured {
		d.Set("value", rvRes.Value)
	} else {
		d.Set("value", d.Get("value").(string))
	}

	return nil
}

func resourceRepositoryVariableUpdate(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return err
	}

	rvcr := newRepositoryVariableFromResource(d)

	_, _, err = pipeApi.UpdateRepositoryPipelineVariable(c.AuthContext, rvcr, workspace, repoSlug, d.Get("uuid").(string))
	if err != nil {
		return fmt.Errorf("error updating Repository Variable (%s): %w", d.Id(), err)
	}

	return resourceRepositoryVariableRead(d, m)
}

func resourceRepositoryVariableDelete(d *schema.ResourceData, m interface{}) error {
	c := m.(Clients).genClient
	pipeApi := c.ApiClient.PipelinesApi

	repo := d.Get("repository").(string)
	workspace, repoSlug, err := repoVarId(repo)
	if err != nil {
		return err
	}

	_, err = pipeApi.DeleteRepositoryPipelineVariable(c.AuthContext, workspace, repoSlug, d.Get("uuid").(string))
	if err != nil {
		return fmt.Errorf("error deleting Repository Variable (%s): %w", d.Id(), err)
	}

	return nil
}

func repoVarId(repo string) (string, string, error) {
	idparts := strings.Split(repo, "/")
	if len(idparts) == 2 {
		return idparts[0], idparts[1], nil
	} else {
		return "", "", fmt.Errorf("incorrect ID format, should match `owner/key`")
	}
}
