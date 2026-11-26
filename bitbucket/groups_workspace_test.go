package bitbucket

import (
	"fmt"
	"io"
	"net/http"
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

// workspaceTransport answers GET 2.0/workspaces/{key} from a map of request
// paths to bodies, answers 404 for any other path, and records every path it
// is asked for.
type workspaceTransport struct {
	bodies   map[string]string
	requests []string
}

func (w *workspaceTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	w.requests = append(w.requests, r.Method+" "+r.URL.EscapedPath())
	body, ok := w.bodies[r.URL.EscapedPath()]
	status := http.StatusOK
	if !ok {
		status, body = http.StatusNotFound, `{"type":"error","error":{"message":"not found"}}`
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func TestGroupsWorkspace(t *testing.T) {
	for _, tc := range []struct {
		name      string
		workspace string
		bodies    map[string]string
		want      string
		requests  []string
		wantErr   string
	}{
		{
			name:      "lowercase slug unchanged without a request",
			workspace: "myws",
			want:      "myws",
		},
		{
			// The 1.0 groups API answers 400 to reads with a mixed-case slug.
			name:      "mixed-case slug lowercased without a request",
			workspace: "MyWorkspace",
			want:      "myworkspace",
		},
		{
			name:      "slug with dash and underscore",
			workspace: "My-Team_1",
			want:      "my-team_1",
		},
		{
			name:      "UUID resolved through the 2.0 workspaces API",
			workspace: "{ws-uuid}",
			bodies:    map[string]string{"/2.0/workspaces/%7Bws-uuid%7D": `{"type":"workspace","uuid":"{ws-uuid}","slug":"myws"}`},
			want:      "myws",
			requests:  []string{"GET /2.0/workspaces/%7Bws-uuid%7D"},
		},
		{
			name:      "UUID response without a slug",
			workspace: "{ws-uuid}",
			bodies:    map[string]string{"/2.0/workspaces/%7Bws-uuid%7D": `{"type":"workspace","uuid":"{ws-uuid}"}`},
			requests:  []string{"GET /2.0/workspaces/%7Bws-uuid%7D"},
			wantErr:   "the response has no slug",
		},
		{
			name:      "unknown UUID",
			workspace: "{missing}",
			requests:  []string{"GET /2.0/workspaces/%7Bmissing%7D"},
			wantErr:   "resolving workspace {missing} to its slug",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rt := &workspaceTransport{bodies: tc.bodies}
			client := clientsWithTransport(rt).httpClient

			got, err := groupsWorkspace(client, tc.workspace)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got %q, %v; want an error containing %q", got, err, tc.wantErr)
				}
			} else if err != nil || got != tc.want {
				t.Fatalf("got %q, %v; want %q", got, err, tc.want)
			}
			if fmt.Sprint(rt.requests) != fmt.Sprint(tc.requests) {
				t.Fatalf("requests %v; want %v", rt.requests, tc.requests)
			}
		})
	}
}
