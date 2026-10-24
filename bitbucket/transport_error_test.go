package bitbucket

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection reset by peer")
}

type notFoundTransport struct{}

func (notFoundTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"type":"error","error":{"message":"Repository not found"}}`)),
		Request:    r,
	}, nil
}

func clientsWithTransport(rt http.RoundTripper) Clients {
	httpClient := &http.Client{Transport: rt}
	conf := bitbucket.NewConfiguration()
	conf.HTTPClient = httpClient
	return Clients{
		genClient:  ProviderConfig{ApiClient: bitbucket.NewAPIClient(conf), AuthContext: context.Background()},
		httpClient: Client{HTTPClient: httpClient},
	}
}

func TestReadRemovesResourceOnNotFound(t *testing.T) {
	clients := clientsWithTransport(notFoundTransport{})
	for _, tc := range []struct {
		name string
		res  *schema.Resource
		id   string
	}{
		{"default_reviewers", resourceDefaultReviewers(), "ws/repo/reviewers"},
		{"project_default_reviewers", resourceProjectDefaultReviewers(), "ws/PRJ"},
	} {
		res := tc.res
		t.Run(tc.name, func(t *testing.T) {
			d := res.TestResourceData()
			d.SetId(tc.id)
			if diags := res.ReadWithoutTimeout(context.Background(), d, clients); diags.HasError() {
				t.Fatalf("expected no error, got %v", diags)
			}
			if d.Id() != "" {
				t.Fatalf("expected resource to be removed from state")
			}
		})
	}
}

func TestReadReturnsErrorOnTransportFailure(t *testing.T) {
	clients := clientsWithTransport(failingTransport{})

	cases := []struct {
		name  string
		res   *schema.Resource
		id    string
		attrs map[string]interface{}
	}{
		{"hook", resourceHook(), "{hook-uuid}", map[string]interface{}{"owner": "ws", "repository": "repo"}},
		{"workspace_hook", resourceWorkspaceHook(), "{hook-uuid}", map[string]interface{}{"workspace": "ws"}},
		{"group", resourceGroup(), "ws/grp", nil},
		{"group_membership", resourceGroupMembership(), "ws/grp/{user}", nil},
		{"branching_model", resourceBranchingModel(), "ws/repo", nil},
		{"project_branching_model", resourceProjectBranchingModel(), "ws/PRJ", nil},
		{"project_user_permission", resourceProjectUserPermission(), "ws:PRJ:{user}", nil},
		{"project_group_permission", resourceProjectGroupPermission(), "ws:PRJ:grp", nil},
		{"repository_user_permission", resourceRepositoryUserPermission(), "ws:repo:{user}", nil},
		{"repository_group_permission", resourceRepositoryGroupPermission(), "ws:repo:grp", nil},
		{"default_reviewers", resourceDefaultReviewers(), "ws/repo/reviewers", nil},
		{"project_default_reviewers", resourceProjectDefaultReviewers(), "ws/PRJ", nil},
		{"repository", resourceRepository(), "ws/repo", nil},
		{"forked_repository", resourceForkedRepository(), "ws/repo", nil},
		{"data_group", dataGroup(), "", map[string]interface{}{"workspace": "ws", "slug": "grp"}},
		{"data_groups", dataGroups(), "", map[string]interface{}{"workspace": "ws"}},
		{"data_group_members", dataGroupMembers(), "", map[string]interface{}{"workspace": "ws", "slug": "grp"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.res.TestResourceData()
			d.SetId(tc.id)
			for k, v := range tc.attrs {
				if err := d.Set(k, v); err != nil {
					t.Fatalf("set %s: %s", k, err)
				}
			}

			read := tc.res.ReadWithoutTimeout
			if read == nil {
				read = tc.res.ReadContext
			}

			diags := read(context.Background(), d, clients)
			if !diags.HasError() || !strings.Contains(diags[0].Summary, "connection reset by peer") {
				t.Fatalf("expected the transport error, got %v", diags)
			}
			if d.Id() != tc.id {
				t.Fatalf("resource was removed from state on a transport error")
			}
		})
	}
}
