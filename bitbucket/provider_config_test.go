package bitbucket

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// fakeAPIServer is a real HTTP server (unlike fakeBitbucket, which replaces
// the transport), so tests can check that requests go to api_url. It serves canned responses by request path and records the
// requests it receives.
type fakeAPIServer struct {
	*httptest.Server
	mu       sync.Mutex
	requests []string
	auth     []string
}

func newFakeAPIServer(t *testing.T, routes map[string]string) *fakeAPIServer {
	t.Helper()
	f := &fakeAPIServer{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		user, _, _ := r.BasicAuth()
		f.auth = append(f.auth, user)
		f.mu.Unlock()

		body, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"type":"error","error":{"message":"not found"}}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	t.Cleanup(f.Close)
	// The provider block sets basic auth; keep other credentials out.
	for _, k := range []string{"BITBUCKET_USERNAME", "BITBUCKET_PASSWORD", "BITBUCKET_OAUTH_TOKEN", "BITBUCKET_ACCESS_TOKEN", "BITBUCKET_OAUTH_CLIENT_ID", "BITBUCKET_OAUTH_CLIENT_SECRET", "BITBUCKET_API_URL", "BITBUCKET_TIMEOUT", "BITBUCKET_MAX_RETRIES"} {
		t.Setenv(k, "")
	}
	return f
}

func (f *fakeAPIServer) seen(req string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.requests {
		if r == req {
			return true
		}
	}
	return false
}

// testProviderConfig points the provider at a fake Bitbucket.
func testProviderConfig(apiURL, extra string) string {
	return fmt.Sprintf(`
provider "bitbucket" {
  api_url  = %q
  username = "terraform"
  password = "secret"
  %s
}
`, apiURL, extra)
}

// api_url is used by both the generated client (GET /2.0/user) and the raw
// HTTP client (GET /2.0/user/emails), which bitbucket_current_user calls.
func TestProvider_apiURL(t *testing.T) {
	fake := newFakeAPIServer(t, map[string]string{
		"GET /2.0/user":        `{"type":"user","uuid":"{6c1f}","username":"terraform","display_name":"Terraform"}`,
		"GET /2.0/user/emails": `{"values":[{"email":"tf@example.com","is_primary":true,"is_confirmed":true}]}`,
	})

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(fake.URL, "") + `
data "bitbucket_current_user" "me" {}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.bitbucket_current_user.me", "uuid", "{6c1f}"),
					resource.TestCheckResourceAttr("data.bitbucket_current_user.me", "display_name", "Terraform"),
					resource.TestCheckResourceAttr("data.bitbucket_current_user.me", "email.0.email", "tf@example.com"),
					func(*terraform.State) error {
						for _, req := range []string{"GET /2.0/user", "GET /2.0/user/emails"} {
							if !fake.seen(req) {
								return fmt.Errorf("fake Bitbucket did not receive %s", req)
							}
						}
						return nil
					},
				),
			},
		},
	})
}

// The api_url works with a trailing slash too (the test above has none).
func TestProvider_apiURLWithTrailingSlash(t *testing.T) {
	fake := newFakeAPIServer(t, map[string]string{
		"GET /2.0/user":        `{"type":"user","uuid":"{6c1f}","username":"terraform"}`,
		"GET /2.0/user/emails": `{"values":[]}`,
	})

	resource.UnitTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfig(fake.URL+"/", "") + `
data "bitbucket_current_user" "me" {}
`,
				Check: resource.TestCheckResourceAttr("data.bitbucket_current_user.me", "uuid", "{6c1f}"),
			},
		},
	})
}

func TestProvider_invalidSettings(t *testing.T) {
	cases := map[string]struct {
		extra string
		want  string
	}{
		"api_url":          {`api_url = "not a url"`, `expected "api_url" to have a host`},
		"timeout format":   {`timeout = "soon"`, `must be a duration`},
		"timeout not > 0":  {`timeout = "0s"`, `must be positive`},
		"negative retries": {`max_retries = -1`, `expected max_retries to be at least \(0\)`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProviderFactories: testAccProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: fmt.Sprintf(`
provider "bitbucket" {
  username = "terraform"
  password = "secret"
  %s
}

data "bitbucket_current_user" "me" {}
`, tc.extra),
						ExpectError: regexp.MustCompile(tc.want),
					},
				},
			})
		})
	}
}

func TestNewHTTPClientTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })

	start := time.Now()
	_, err := newHTTPClient(100*time.Millisecond, 0).Get(srv.URL)
	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("timeout took %s", elapsed)
	}
}

func TestNewHTTPClientMaxRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	resp, err := newHTTPClient(defaultHTTPTimeout, 1).Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if calls != 2 {
		t.Fatalf("max_retries = 1: got %d calls, want 2", calls)
	}
}
