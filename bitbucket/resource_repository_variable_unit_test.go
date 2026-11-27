package bitbucket

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/DrFaust92/bitbucket-go-client"
)

// repositoryVariablesServer serves `total` variables KEY_1..KEY_n, 10 per page, like the variables API.
func repositoryVariablesServer(t *testing.T, total int, calls *int32) *httptest.Server {
	const pageSize = 10

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(calls, 1)
		if want := "/repositories/workspace/repo/pipelines_config/variables"; strings.TrimSuffix(r.URL.Path, "/") != want {
			t.Errorf("unexpected request path %q, want %q", r.URL.Path, want)
		}

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			_, _ = fmt.Sscanf(p, "%d", &page)
		}

		var values []string
		for i := (page-1)*pageSize + 1; i <= total && i <= page*pageSize; i++ {
			values = append(values, fmt.Sprintf(`{"uuid":"{var-%d}","key":"KEY_%d","secured":true}`, i, i))
		}

		next := ""
		if page*pageSize < total {
			next = fmt.Sprintf(`,"next":"%s%s?page=%d"`, "http://"+r.Host, r.URL.Path, page+1)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"size":%d,"page":%d,"pagelen":%d,"values":[%s]%s}`, total, page, pageSize, strings.Join(values, ","), next)
	}))
}

func testRepositoryVariableClients(url string) Clients {
	conf := bitbucket.NewConfiguration()
	conf.BasePath = url
	return Clients{genClient: ProviderConfig{ApiClient: bitbucket.NewAPIClient(conf), AuthContext: context.Background()}}
}

func TestRepositoryVariableImportByKey(t *testing.T) {
	var calls int32
	server := repositoryVariablesServer(t, 12, &calls)
	defer server.Close()
	clients := testRepositoryVariableClients(server.URL)

	d := resourceRepositoryVariable().TestResourceData()
	d.SetId("workspace/repo/KEY_12")

	got, err := resourceRepositoryVariableImport(context.Background(), d, clients)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d resources, want 1", len(got))
	}
	if d.Id() != "KEY_12" {
		t.Errorf("id = %q, want KEY_12", d.Id())
	}
	if uuid := d.Get("uuid").(string); uuid != "{var-12}" {
		t.Errorf("uuid = %q, want {var-12} from page 2", uuid)
	}
	if repo := d.Get("repository").(string); repo != "workspace/repo" {
		t.Errorf("repository = %q, want workspace/repo", repo)
	}
	if calls != 2 {
		t.Errorf("made %d API calls, want 2 (one per page)", calls)
	}
}

func TestRepositoryVariableImportByKeyNotFound(t *testing.T) {
	var calls int32
	server := repositoryVariablesServer(t, 12, &calls)
	defer server.Close()

	d := resourceRepositoryVariable().TestResourceData()
	d.SetId("workspace/repo/MISSING")

	_, err := resourceRepositoryVariableImport(context.Background(), d, testRepositoryVariableClients(server.URL))
	if err == nil || !strings.Contains(err.Error(), `"MISSING" not found in workspace/repo`) {
		t.Fatalf("err = %v, want a not found error", err)
	}
}

func TestRepositoryVariableImportByUUID(t *testing.T) {
	var calls int32
	server := repositoryVariablesServer(t, 12, &calls)
	defer server.Close()

	d := resourceRepositoryVariable().TestResourceData()
	d.SetId("workspace/repo/KEY_3/{var-3}")

	if _, err := resourceRepositoryVariableImport(context.Background(), d, testRepositoryVariableClients(server.URL)); err != nil {
		t.Fatalf("import failed: %v", err)
	}
	if d.Id() != "KEY_3" || d.Get("uuid").(string) != "{var-3}" || d.Get("repository").(string) != "workspace/repo" {
		t.Errorf("got id=%q uuid=%q repository=%q", d.Id(), d.Get("uuid"), d.Get("repository"))
	}
	if calls != 0 {
		t.Errorf("made %d API calls, want none when the UUID is given", calls)
	}
}

func TestRepositoryVariableImportInvalidID(t *testing.T) {
	for _, id := range []string{"workspace/repo", "workspace//KEY", "workspace/repo/KEY/", "a/b/c/d/e"} {
		d := resourceRepositoryVariable().TestResourceData()
		d.SetId(id)
		if _, err := resourceRepositoryVariableImport(context.Background(), d, Clients{}); err == nil {
			t.Errorf("ID %q: expected a format error", id)
		}
	}
}
