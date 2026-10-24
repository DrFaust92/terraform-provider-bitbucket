package bitbucket

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type cursorPagesTransport struct {
	pages map[string]string
	seen  []string
}

func (c *cursorPagesTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	key := r.URL.Path
	if p := r.URL.Query().Get("page"); p != "" {
		key += "?page=" + p
	}
	c.seen = append(c.seen, key)
	body, ok := c.pages[key]
	status := http.StatusOK
	if !ok {
		status, body = http.StatusBadRequest, `{"type":"error","error":{"message":"Invalid page"}}`
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func TestDefaultReviewersReadFollowsCursorPages(t *testing.T) {
	rt := &cursorPagesTransport{pages: map[string]string{
		"/2.0/repositories/ws/repo/default-reviewers":               `{"values":[{"uuid":"{a}"}],"next":"https://api.bitbucket.org/2.0/repositories/ws/repo/default-reviewers?page=Zm9vYmFy"}`,
		"/2.0/repositories/ws/repo/default-reviewers?page=Zm9vYmFy": `{"values":[{"uuid":"{b}"}]}`,
	}}

	res := resourceDefaultReviewers()
	d := res.TestResourceData()
	d.SetId("ws/repo/reviewers")
	if diags := res.ReadWithoutTimeout(context.Background(), d, clientsWithTransport(rt)); diags.HasError() {
		t.Fatalf("unexpected error: %v (requests: %v)", diags, rt.seen)
	}
	if got := d.Get("reviewers.#"); got != 2 {
		t.Fatalf("got %v reviewers, want 2 (requests: %v)", got, rt.seen)
	}
}

func TestProjectDefaultReviewersReadFollowsCursorPages(t *testing.T) {
	rt := &cursorPagesTransport{pages: map[string]string{
		"/2.0/workspaces/ws/projects/PRJ/default-reviewers":               `{"values":[{"user":{"uuid":"{a}"}}],"next":"https://api.bitbucket.org/2.0/workspaces/ws/projects/PRJ/default-reviewers?page=Y3Vyc29y"}`,
		"/2.0/workspaces/ws/projects/PRJ/default-reviewers?page=Y3Vyc29y": `{"values":[{"user":{"uuid":"{b}"}}]}`,
	}}

	res := resourceProjectDefaultReviewers()
	d := res.TestResourceData()
	d.SetId("ws/PRJ")
	if diags := res.ReadWithoutTimeout(context.Background(), d, clientsWithTransport(rt)); diags.HasError() {
		t.Fatalf("unexpected error: %v (requests: %v)", diags, rt.seen)
	}
	if got := d.Get("reviewers.#"); got != 2 {
		t.Fatalf("got %v reviewers, want 2 (requests: %v)", got, rt.seen)
	}
}

func TestWorkspaceMembersReadFollowsCursorPages(t *testing.T) {
	rt := &cursorPagesTransport{pages: map[string]string{
		"/2.0/workspaces/ws/members":             `{"values":[{"user":{"uuid":"{a}"}}],"next":"https://api.bitbucket.org/2.0/workspaces/ws/members?page=bmV4dA"}`,
		"/2.0/workspaces/ws/members?page=bmV4dA": `{"values":[{"user":{"uuid":"{b}"}}]}`,
	}}

	res := dataWorkspaceMembers()
	d := res.TestResourceData()
	if err := d.Set("workspace", "ws"); err != nil {
		t.Fatal(err)
	}
	read := res.ReadWithoutTimeout
	if read == nil {
		read = res.ReadContext
	}
	if diags := read(context.Background(), d, clientsWithTransport(rt)); diags.HasError() {
		t.Fatalf("unexpected error: %v (requests: %v)", diags, rt.seen)
	}
	if got := d.Get("members.#"); got != 2 {
		t.Fatalf("got %v members, want 2 (requests: %v)", got, rt.seen)
	}
}
