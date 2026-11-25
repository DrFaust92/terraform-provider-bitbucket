package bitbucket

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// The 1.0 groups API rejects a workspace UUID, so a UUID workspace (such as
// data.bitbucket_workspace.x.id) is resolved to its slug for those calls,
// while state keeps the value from the configuration (#258).
func TestGroup_workspaceUUID(t *testing.T) {
	group := `{"name":"devs","slug":"devs","permission":"read","auto_add":false,"email_forwarding_disabled":false}`
	fake := newFakeAPIServer(t, map[string]string{
		"GET /2.0/workspaces/{ws-uuid}": `{"type":"workspace","uuid":"{ws-uuid}","slug":"myws"}`,
		"POST /1.0/groups/myws":         group,
		"GET /1.0/groups/myws/devs":     group,
		"DELETE /1.0/groups/myws/devs":  ``,
	})
	name := "bitbucket_group.test"

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(fake.URL, "") + `
resource "bitbucket_group" "test" {
  workspace  = "{ws-uuid}"
  name       = "devs"
  permission = "read"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "workspace", "{ws-uuid}"),
					resource.TestCheckResourceAttr(name, "slug", "devs"),
					func(*terraform.State) error {
						fake.mu.Lock()
						defer fake.mu.Unlock()
						for _, r := range fake.requests {
							if strings.Contains(r, "/1.0/groups/{") {
								return fmt.Errorf("groups API called with the workspace UUID: %s", r)
							}
						}
						return nil
					},
				),
			},
		},
	})
}

func TestGroupsWorkspace_slugPassesThrough(t *testing.T) {
	got, err := groupsWorkspace(Client{BaseURL: "http://127.0.0.1:1/"}, "myws")
	if err != nil || got != "myws" {
		t.Fatalf("got %q, %v; want the slug unchanged without a request", got, err)
	}
}
