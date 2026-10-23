package bitbucket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DrFaust92/bitbucket-go-client"
)

func TestDeploymentVariableReadFollowsPages(t *testing.T) {
	const pageSize = 10
	const total = 12

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			_, _ = fmt.Sscanf(p, "%d", &page)
		}

		var values []string
		for i := (page-1)*pageSize + 1; i <= total && i <= page*pageSize; i++ {
			values = append(values, fmt.Sprintf(`{"uuid":"{var-%d}","key":"KEY_%d","value":"v","secured":false}`, i, i))
		}

		next := ""
		if page*pageSize < total {
			next = fmt.Sprintf(`,"next":"%s%s?page=%d"`, "http://"+r.Host, r.URL.Path, page+1)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"size":%d,"page":%d,"pagelen":%d,"values":[%s]%s}`, total, page, pageSize, strings.Join(values, ","), next)
	}))
	defer server.Close()

	conf := bitbucket.NewConfiguration()
	conf.BasePath = server.URL
	clients := Clients{genClient: ProviderConfig{ApiClient: bitbucket.NewAPIClient(conf), AuthContext: context.Background()}}

	d := resourceDeploymentVariable().TestResourceData()
	d.SetId("{var-12}")
	if err := d.Set("deployment", "workspace/repo:{env}"); err != nil {
		t.Fatal(err)
	}

	if diags := resourceDeploymentVariableRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read failed: %v", diags)
	}

	if d.Id() != "{var-12}" {
		t.Fatalf("variable on page 2 was dropped from state")
	}
	if got := d.Get("key").(string); got != "KEY_12" {
		t.Errorf("key = %q, want KEY_12", got)
	}

	// A variable that doesn't exist on any page is removed from state.
	d.SetId("{missing}")
	if diags := resourceDeploymentVariableRead(context.Background(), d, clients); diags.HasError() {
		t.Fatalf("read failed: %v", diags)
	}
	if d.Id() != "" {
		t.Errorf("missing variable should be removed from state")
	}
}

func TestHandleClientErrorReturnsTransportErrors(t *testing.T) {
	err := errors.New("connection reset by peer")
	if got := handleClientError(nil, err); !errors.Is(got, err) {
		t.Errorf("handleClientError(nil, err) = %v, want %v", got, err)
	}
	if got := handleClientError(nil, nil); got != nil {
		t.Errorf("handleClientError(nil, nil) = %v, want nil", got)
	}
}

func TestExtractErrorMessage(t *testing.T) {
	cases := map[string]string{
		`{"type":"error","error":{"message":"Bad request","detail":"The request body contains invalid properties","data":{"arguments":{"secured-required":"Property secured is required"}}}}`: `Bad request: The request body contains invalid properties {"arguments":{"secured-required":"Property secured is required"}}`,
		`{"type":"error","error":{"message":"Repository not found"}}`:                                                                              "Repository not found",
		`{"type":"error","error":{"message":"Your credentials lack scopes","detail":{"required":["pipeline:variable"],"granted":["repository"]}}}`: "Your credentials lack scopes Required: [pipeline:variable] Granted: [repository]",
		`{"type":"error"}`:  `{"type":"error"}`,
		`<html>oops</html>`: `<html>oops</html>`,
	}

	for body, want := range cases {
		if got := extractErrorMessage([]byte(body)); got != want {
			t.Errorf("extractErrorMessage(%s)\n got: %s\nwant: %s", body, got, want)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientErrorIncludesDetail(t *testing.T) {
	c := &Client{HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"message":"Bad request","detail":"name is too long"}}`)),
			Request:    r,
		}, nil
	})}}

	_, err := c.Get("2.0/thing")
	if err == nil || !strings.Contains(err.Error(), "Bad request: name is too long") {
		t.Errorf("expected error detail, got %v", err)
	}
}

func TestClientDoTransportErrorDoesNotPanic(t *testing.T) {
	transportErr := errors.New("connection reset by peer")
	c := &Client{HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, transportErr
	})}}

	if _, err := c.Get("2.0/thing"); !errors.Is(err, transportErr) {
		t.Fatalf("expected the transport error, got %v", err)
	}
}

func TestSshKeyBody(t *testing.T) {
	cases := map[string]string{
		"ssh-ed25519 AAAAC3Nza user@host":      "ssh-ed25519 AAAAC3Nza",
		"ssh-ed25519 AAAAC3Nza":                "ssh-ed25519 AAAAC3Nza",
		"  ssh-rsa AAAAB3Nza  a comment here ": "ssh-rsa AAAAB3Nza",
		"":                                     "",
	}

	for in, want := range cases {
		if got := sshKeyBody(in); got != want {
			t.Errorf("sshKeyBody(%q) = %q, want %q", in, got, want)
		}
	}
}
