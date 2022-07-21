package bitbucket

import (
	"encoding/json"
	"fmt"

	"github.com/migara/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataWorkspaceMembers() *schema.Resource {
	return &schema.Resource{
		Read: dataReadWorkspaceMembers,

		Schema: map[string]*schema.Schema{
			"workspace": {
				Type:     schema.TypeString,
				Required: true,
			},
			"members": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func dataReadWorkspaceMembers(d *schema.ResourceData, m interface{}) error {
	client := m.(Clients).httpClient

	workspace := d.Get("workspace").(string)
	resourceURL := fmt.Sprintf("2.0/workspaces/%s/members", workspace)

	_, err := client.Get(resourceURL)
	if err != nil {
		return err
	}

	var paginatedMemberships bitbucket.PaginatedWorkspaceMemberships
	var members []string

	for {
		membersRes, err := client.Get(resourceURL)
		if err != nil {
			return err
		}

		decoder := json.NewDecoder(membersRes.Body)
		err = decoder.Decode(&paginatedMemberships)
		if err != nil {
			return err
		}

		for _, member := range paginatedMemberships.Values {
			members = append(members, member.User.Uuid)
		}

		if paginatedMemberships.Next != "" {
			nextPage := paginatedMemberships.Page + 1
			resourceURL = fmt.Sprintf("2.0/workspaces/%s/members?page=%d", workspace, nextPage)
			paginatedMemberships = bitbucket.PaginatedWorkspaceMemberships{}
		} else {
			break
		}
	}

	d.SetId(workspace)
	d.Set("workspace", workspace)
	d.Set("members", members)

	return nil
}
