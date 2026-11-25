package bitbucket

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataWorkspace_newAttributes(t *testing.T) {
	fake := newFakeAPIServer(t, map[string]string{
		"GET /2.0/workspaces/ws": `{"type":"workspace","uuid":"{ws-1}","slug":"ws","name":"WS","is_private":true,
			"is_personal":true,"is_privacy_enforced":true,"forking_mode":"internal_only","enforced_signed_commits":true}`,
	})
	name := "data.bitbucket_workspace.ws"

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(fake.URL, "") + `
data "bitbucket_workspace" "ws" {
  workspace = "ws"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "is_personal", "true"),
					resource.TestCheckResourceAttr(name, "is_privacy_enforced", "true"),
					resource.TestCheckResourceAttr(name, "forking_mode", "internal_only"),
					resource.TestCheckResourceAttr(name, "enforced_signed_commits", "true"),
				),
			},
		},
	})
}

func TestDataRepositoryAndProject_enforcedSignedCommits(t *testing.T) {
	fake := newFakeAPIServer(t, map[string]string{
		"GET /2.0/repositories/ws/repo":     `{"type":"repository","uuid":"{repo-1}","slug":"repo","name":"repo","full_name":"ws/repo","enforced_signed_commits":true}`,
		"GET /2.0/workspaces/ws/projects/P": `{"type":"project","uuid":"{proj-1}","key":"P","name":"P","enforced_signed_commits":true}`,
	})

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(fake.URL, "") + `
data "bitbucket_repository" "repo" {
  workspace = "ws"
  repo_slug = "repo"
}

data "bitbucket_project" "p" {
  workspace = "ws"
  key       = "P"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.bitbucket_repository.repo", "enforced_signed_commits", "true"),
					resource.TestCheckResourceAttr("data.bitbucket_project.p", "enforced_signed_commits", "true"),
				),
			},
		},
	})
}

func TestSshKey_newAttributes(t *testing.T) {
	const key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKqP3Cr632C2dNhhgKVcon4ldUSAeKiku2yP9O9/bDtY"
	body := `{"type":"ssh_key","uuid":"{key-1}","key":"` + key + `","label":"laptop",
		"fingerprint":"SHA256:abc","expires_on":"2027-01-02T03:04:05+00:00","last_used":"2026-11-20T10:00:00.123456+00:00"}`
	fake := newFakeAPIServer(t, map[string]string{
		"POST /2.0/users/u1/ssh-keys":           body,
		"GET /2.0/users/u1/ssh-keys/{key-1}":    body,
		"DELETE /2.0/users/u1/ssh-keys/{key-1}": ``,
	})
	name := "bitbucket_ssh_key.test"

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(fake.URL, "") + `
resource "bitbucket_ssh_key" "test" {
  user  = "u1"
  key   = "` + key + `"
  label = "laptop"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "fingerprint", "SHA256:abc"),
					resource.TestCheckResourceAttr(name, "expires_on", "2027-01-02T03:04:05Z"),
					resource.TestCheckResourceAttr(name, "last_used", "2026-11-20T10:00:00Z"),
				),
			},
		},
	})
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(time.Time{}); got != "" {
		t.Errorf("zero time: got %q, want empty", got)
	}
	ts := time.Date(2027, 1, 2, 3, 4, 5, 0, time.FixedZone("x", 3600))
	if got := formatTime(ts); got != "2027-01-02T02:04:05Z" {
		t.Errorf("got %q", got)
	}
}
