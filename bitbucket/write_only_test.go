package bitbucket

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// fakeBitbucket is a minimal in-memory Bitbucket API for repository variables and hooks. It
// records the secret values it received so tests can check what was sent.
type fakeBitbucket struct {
	mu        sync.Mutex
	variables map[string]map[string]interface{}
	hooks     map[string]map[string]interface{}
	received  []string
	nextID    int
}

func newFakeBitbucket() *fakeBitbucket {
	return &fakeBitbucket{variables: map[string]map[string]interface{}{}, hooks: map[string]map[string]interface{}{}}
}

func (f *fakeBitbucket) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var body map[string]interface{}
	if r.Body != nil {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
	}

	const varsPath = "/2.0/repositories/ws/repo/pipelines_config/variables"
	const hooksPath = "/2.0/repositories/ws/repo/hooks"
	path := strings.TrimSuffix(r.URL.Path, "/")

	store, secretField, base := f.variables, "value", varsPath
	if strings.HasPrefix(path, hooksPath) {
		store, secretField, base = f.hooks, "secret", hooksPath
	} else if !strings.HasPrefix(path, varsPath) {
		return jsonResponse(r, http.StatusNotFound, map[string]interface{}{"type": "error", "error": map[string]string{"message": "unexpected path " + path}}), nil
	}

	id := strings.TrimPrefix(strings.TrimPrefix(path, base), "/")

	switch {
	case r.Method == http.MethodPost && id == "":
		f.nextID++
		id = fmt.Sprintf("{%d}", f.nextID)
		body["uuid"] = id
		f.received = append(f.received, fmt.Sprint(body[secretField]))
		store[id] = body
	case r.Method == http.MethodPut && store[id] != nil:
		body["uuid"] = id
		f.received = append(f.received, fmt.Sprint(body[secretField]))
		store[id] = body
	case r.Method == http.MethodDelete && store[id] != nil:
		delete(store, id)
		return jsonResponse(r, http.StatusNoContent, nil), nil
	case r.Method == http.MethodGet && store[id] != nil:
	default:
		return jsonResponse(r, http.StatusNotFound, map[string]interface{}{"type": "error", "error": map[string]string{"message": "not found"}}), nil
	}

	out := map[string]interface{}{}
	for k, v := range store[id] {
		out[k] = v
	}
	// Bitbucket never returns hook secrets or secured variable values.
	if secretField == "secret" {
		out["secret_set"] = out["secret"] != ""
		delete(out, "secret")
	} else if out["secured"] == true {
		delete(out, "value")
	}

	return jsonResponse(r, http.StatusOK, out), nil
}

func jsonResponse(r *http.Request, status int, v interface{}) *http.Response {
	var body []byte
	if v != nil {
		body, _ = json.Marshal(v)
	}

	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
		Request:    r,
	}
}

func withFakeBitbucket(t *testing.T) *fakeBitbucket {
	t.Helper()
	fake := newFakeBitbucket()

	original := http.DefaultTransport
	http.DefaultTransport = fake
	t.Cleanup(func() { http.DefaultTransport = original })

	t.Setenv("BITBUCKET_USERNAME", "user")
	t.Setenv("BITBUCKET_PASSWORD", "password")
	for _, k := range []string{"BITBUCKET_OAUTH_TOKEN", "BITBUCKET_OAUTH_CLIENT_ID", "BITBUCKET_OAUTH_CLIENT_SECRET"} {
		t.Setenv(k, "")
	}

	return fake
}

func (f *fakeBitbucket) lastReceived() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.received) == 0 {
		return ""
	}
	return f.received[len(f.received)-1]
}

func testCheckReceived(f *fakeBitbucket, want string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if got := f.lastReceived(); got != want {
			return fmt.Errorf("API received secret %q, want %q", got, want)
		}
		return nil
	}
}

// testCheckSecretNotInState fails if secret appears in any attribute of any resource in state.
func testCheckSecretNotInState(secret string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, rs := range s.RootModule().Resources {
			for k, v := range rs.Primary.Attributes {
				if strings.Contains(v, secret) {
					return fmt.Errorf("secret found in state: %s.%s", name, k)
				}
			}
		}
		return nil
	}
}

func TestWriteOnlyRepositoryVariable(t *testing.T) {
	fake := withFakeBitbucket(t)
	resourceName := "bitbucket_repository_variable.test"

	config := func(value, version string) string {
		return fmt.Sprintf(`
resource "bitbucket_repository_variable" "test" {
  repository       = "ws/repo"
  key              = "TOKEN"
  value_wo         = %q
  value_wo_version = %q
  secured          = false
}
`, value, version)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: writeOnlyTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("s3cret", "1"),
				Check: resource.ComposeTestCheckFunc(
					testCheckReceived(fake, "s3cret"),
					// The API returns unsecured values, they must still not end up in state.
					testCheckSecretNotInState("s3cret"),
					resource.TestCheckResourceAttr(resourceName, "value", ""),
					resource.TestCheckNoResourceAttr(resourceName, "value_wo"),
					resource.TestCheckResourceAttr(resourceName, "value_wo_version", "1"),
				),
			},
			{
				Config: config("rotated", "2"),
				Check: resource.ComposeTestCheckFunc(
					testCheckReceived(fake, "rotated"),
					testCheckSecretNotInState("rotated"),
					resource.TestCheckResourceAttr(resourceName, "value_wo_version", "2"),
				),
			},
		},
	})
}

func TestWriteOnlyHookSecret(t *testing.T) {
	fake := withFakeBitbucket(t)
	resourceName := "bitbucket_hook.test"

	config := func(secret, version string) string {
		return fmt.Sprintf(`
resource "bitbucket_hook" "test" {
  owner             = "ws"
  repository        = "repo"
  url               = "https://example.com/hook"
  description       = "test"
  events            = ["repo:push"]
  secret_wo         = %q
  secret_wo_version = %q
}
`, secret, version)
	}

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: writeOnlyTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("hook-secret", "1"),
				Check: resource.ComposeTestCheckFunc(
					testCheckReceived(fake, "hook-secret"),
					testCheckSecretNotInState("hook-secret"),
					resource.TestCheckResourceAttr(resourceName, "secret", ""),
					resource.TestCheckNoResourceAttr(resourceName, "secret_wo"),
					resource.TestCheckResourceAttr(resourceName, "secret_set", "true"),
				),
			},
			{
				Config: config("hook-secret-2", "2"),
				Check:  testCheckReceived(fake, "hook-secret-2"),
			},
		},
	})
}

func TestWriteOnlyConflictsWithValue(t *testing.T) {
	withFakeBitbucket(t)

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: writeOnlyTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "bitbucket_repository_variable" "test" {
  repository       = "ws/repo"
  key              = "TOKEN"
  value            = "plain"
  value_wo         = "secret"
  value_wo_version = "1"
}
`,
				ExpectError: regexp.MustCompile(`conflicts with value_wo|"value": only one of`),
			},
			{
				Config: `
resource "bitbucket_repository_variable" "test" {
  repository = "ws/repo"
  key        = "TOKEN"
  value_wo   = "secret"
}
`,
				ExpectError: regexp.MustCompile(`all of .value_wo,value_wo_version. must be specified`),
			},
		},
	})
}

var writeOnlyTestProviderFactories = map[string]func() (*schema.Provider, error){
	"bitbucket": func() (*schema.Provider, error) { return Provider(), nil },
}
