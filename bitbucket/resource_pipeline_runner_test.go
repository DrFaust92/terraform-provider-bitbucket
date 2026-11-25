package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// fakeRunnersAPI is an in-memory Bitbucket Pipelines runners API, served over
// HTTP so the provider reaches it through api_url.
type fakeRunnersAPI struct {
	*httptest.Server
	mu      sync.Mutex
	runners map[string]map[string]interface{} // by collection path + "/" + uuid
	nextID  int
	puts    int
}

func newFakeRunnersAPI(t *testing.T) *fakeRunnersAPI {
	t.Helper()
	f := &fakeRunnersAPI{runners: map[string]map[string]interface{}{}}
	f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.Close)
	for _, k := range []string{"BITBUCKET_USERNAME", "BITBUCKET_PASSWORD", "BITBUCKET_OAUTH_TOKEN", "BITBUCKET_ACCESS_TOKEN", "BITBUCKET_OAUTH_CLIENT_ID", "BITBUCKET_OAUTH_CLIENT_SECRET", "BITBUCKET_API_URL"} {
		t.Setenv(k, "")
	}
	return f
}

var runnersCollection = regexp.MustCompile(`^/2\.0/(workspaces/[^/]+|repositories/[^/]+/[^/]+)/pipelines-config/runners$`)

func (f *fakeRunnersAPI) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	writeJSON := func(status int, v interface{}) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if v != nil {
			_ = json.NewEncoder(w).Encode(v)
		}
	}
	notFound := func() {
		writeJSON(http.StatusNotFound, map[string]interface{}{"type": "error", "error": map[string]string{"message": "not found"}})
	}

	path := r.URL.Path
	if r.Method == http.MethodPost && runnersCollection.MatchString(path) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.nextID++
		uuid := fmt.Sprintf("{runner-%d}", f.nextID)
		runner := map[string]interface{}{
			"type":   "pipeline_runner",
			"uuid":   uuid,
			"name":   body["name"],
			"labels": body["labels"],
			"state":  map[string]interface{}{"status": "UNREGISTERED", "cordoned": false},
			"oauth_client": map[string]interface{}{
				"id":             fmt.Sprintf("client-%d", f.nextID),
				"token_endpoint": "https://auth.example.com/oauth/token",
				"audience":       "api.atlassian.com",
			},
		}
		f.runners[path+"/"+uuid] = runner
		// Only the create response includes the secret.
		created := map[string]interface{}{}
		for k, v := range runner {
			created[k] = v
		}
		created["oauth_client"] = map[string]interface{}{
			"id":             fmt.Sprintf("client-%d", f.nextID),
			"secret":         fmt.Sprintf("secret-%d", f.nextID),
			"token_endpoint": "https://auth.example.com/oauth/token",
			"audience":       "api.atlassian.com",
		}
		writeJSON(http.StatusOK, created)
		return
	}

	runner, ok := f.runners[path]
	if !ok {
		notFound()
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(http.StatusOK, runner)
	case http.MethodPut:
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		runner["name"], runner["labels"] = body["name"], body["labels"]
		f.puts++
		writeJSON(http.StatusOK, runner)
	case http.MethodDelete:
		delete(f.runners, path)
		w.WriteHeader(http.StatusNoContent)
	default:
		notFound()
	}
}

func (f *fakeRunnersAPI) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.runners)
}

func (f *fakeRunnersAPI) deleteAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runners = map[string]map[string]interface{}{}
}

func testPipelineRunnerConfig(apiURL, repository, name string, labels ...string) string {
	repo := ""
	if repository != "" {
		repo = fmt.Sprintf("repository = %q", repository)
	}
	return testProviderConfig(apiURL, "") + fmt.Sprintf(`
resource "bitbucket_pipeline_runner" "test" {
  workspace = "ws"
  %s
  name      = %q
  labels    = [%s]
}
`, repo, name, `"`+strings.Join(labels, `", "`)+`"`)
}

func TestPipelineRunner_workspace(t *testing.T) {
	fake := newFakeRunnersAPI(t)
	name := "bitbucket_pipeline_runner.test"
	var firstUUID string

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(*terraform.State) error {
			if n := fake.count(); n != 0 {
				return fmt.Errorf("%d runners left after destroy", n)
			}
			return nil
		},
		Steps: []resource.TestStep{
			{
				Config: testPipelineRunnerConfig(fake.URL, "", "build-1", "self.hosted", "linux"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr(name, "id", regexp.MustCompile(`^ws/\{runner-1\}$`)),
					resource.TestCheckResourceAttr(name, "uuid", "{runner-1}"),
					resource.TestCheckResourceAttr(name, "name", "build-1"),
					resource.TestCheckResourceAttr(name, "labels.#", "2"),
					resource.TestCheckTypeSetElemAttr(name, "labels.*", "self.hosted"),
					resource.TestCheckResourceAttr(name, "status", "UNREGISTERED"),
					resource.TestCheckResourceAttr(name, "cordoned", "false"),
					resource.TestCheckResourceAttr(name, "oauth_client_id", "client-1"),
					// The secret from the create response survives the refresh
					// that follows, although reads don't return it.
					resource.TestCheckResourceAttr(name, "oauth_client_secret", "secret-1"),
					resource.TestCheckResourceAttr(name, "oauth_token_endpoint", "https://auth.example.com/oauth/token"),
					resource.TestCheckResourceAttr(name, "oauth_audience", "api.atlassian.com"),
					func(s *terraform.State) error {
						firstUUID = s.RootModule().Resources[name].Primary.Attributes["uuid"]
						return nil
					},
				),
			},
			{
				// Name and labels update in place.
				Config: testPipelineRunnerConfig(fake.URL, "", "build-renamed", "self.hosted", "linux", "large"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", "build-renamed"),
					resource.TestCheckResourceAttr(name, "labels.#", "3"),
					resource.TestCheckResourceAttr(name, "oauth_client_secret", "secret-1"),
					func(s *terraform.State) error {
						if got := s.RootModule().Resources[name].Primary.Attributes["uuid"]; got != firstUUID {
							return fmt.Errorf("runner was replaced: uuid %s, want %s", got, firstUUID)
						}
						if fake.puts != 1 {
							return fmt.Errorf("got %d PUT requests, want 1", fake.puts)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      name,
				ImportState:       true,
				ImportStateVerify: true,
				// Bitbucket returns the secret only on create.
				ImportStateVerifyIgnore: []string{"oauth_client_secret"},
			},
		},
	})
}

func TestPipelineRunner_repository(t *testing.T) {
	fake := newFakeRunnersAPI(t)
	name := "bitbucket_pipeline_runner.test"

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testPipelineRunnerConfig(fake.URL, "repo", "repo-runner", "self.hosted", "linux"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "id", "ws/repo/{runner-1}"),
					resource.TestCheckResourceAttr(name, "repository", "repo"),
					resource.TestCheckResourceAttr(name, "oauth_client_secret", "secret-1"),
				),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth_client_secret"},
			},
		},
	})
}

// A runner deleted outside Terraform is created again.
func TestPipelineRunner_disappears(t *testing.T) {
	fake := newFakeRunnersAPI(t)
	name := "bitbucket_pipeline_runner.test"
	config := testPipelineRunnerConfig(fake.URL, "", "build-1", "self.hosted", "linux")

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "uuid", "{runner-1}"),
					func(*terraform.State) error { fake.deleteAll(); return nil },
				),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(name, "uuid", "{runner-2}"),
			},
		},
	})
}

func TestPipelineRunner_invalidImportID(t *testing.T) {
	fake := newFakeRunnersAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testPipelineRunnerConfig(fake.URL, "", "build-1", "self.hosted"),
				ResourceName:  "bitbucket_pipeline_runner.test",
				ImportState:   true,
				ImportStateId: "not-a-runner-id",
				ExpectError:   regexp.MustCompile(`unexpected pipeline runner ID`),
			},
		},
	})
}

func TestPipelineRunnerID(t *testing.T) {
	cases := []struct {
		id                    string
		workspace, repo, uuid string
		ok                    bool
	}{
		{"ws/{u}", "ws", "", "{u}", true},
		{"ws/repo/{u}", "ws", "repo", "{u}", true},
		{"ws", "", "", "", false},
		{"ws//{u}", "", "", "", false},
		{"a/b/c/d", "", "", "", false},
	}
	for _, tc := range cases {
		w, r, u, err := parsePipelineRunnerID(tc.id)
		if (err == nil) != tc.ok || w != tc.workspace || r != tc.repo || u != tc.uuid {
			t.Errorf("parsePipelineRunnerID(%q) = %q, %q, %q, %v", tc.id, w, r, u, err)
		}
		if tc.ok && pipelineRunnerID(w, r, u) != tc.id {
			t.Errorf("pipelineRunnerID round trip: %q", tc.id)
		}
	}

	if got := pipelineRunnersEndpoint("ws", "", "{u}"); got != "2.0/workspaces/ws/pipelines-config/runners/%7Bu%7D" {
		t.Errorf("workspace endpoint = %s", got)
	}
	if got := pipelineRunnersEndpoint("ws", "repo", ""); got != "2.0/repositories/ws/repo/pipelines-config/runners" {
		t.Errorf("repository endpoint = %s", got)
	}
}

// TestAccBitbucketPipelineRunner_basic runs against a real workspace
// (TF_ACC=1, BITBUCKET_TEAM and credentials).
func TestAccBitbucketPipelineRunner_basic(t *testing.T) {
	workspace := os.Getenv("BITBUCKET_TEAM")
	rName := acctest.RandomWithPrefix("tf-test")
	name := "bitbucket_pipeline_runner.test"
	config := func(runnerName string) string {
		return fmt.Sprintf(`
resource "bitbucket_pipeline_runner" "test" {
  workspace = %q
  name      = %q
  labels    = ["self.hosted", "linux"]
}
`, workspace, runnerName)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(name, "name", rName),
					resource.TestCheckResourceAttrSet(name, "uuid"),
					resource.TestCheckResourceAttrSet(name, "oauth_client_id"),
					resource.TestCheckResourceAttrSet(name, "oauth_client_secret"),
				),
			},
			{
				Config: config(rName + "-renamed"),
				Check:  resource.TestCheckResourceAttr(name, "name", rName+"-renamed"),
			},
			{
				ResourceName:            name,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"oauth_client_secret"},
			},
		},
	})
}
