package bitbucket

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	testDPPWorkspace  = "example-workspace"
	testDPPRepository = "example-repository"
	testDPPID         = testDPPWorkspace + "/" + testDPPRepository
	testDPPPath       = "/internal/repositories/" + testDPPWorkspace + "/" + testDPPRepository + "/pipelines-config/dynamic-pipelines-provider"

	testDPPProductionARI  = "ari:cloud:ecosystem::extension/00000000-0000-0000-0000-000000000001/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/static/example-dynamic-pipelines-provider"
	testDPPDevelopmentARI = "ari:cloud:ecosystem::extension/00000000-0000-0000-0000-000000000001/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/static/example-dynamic-pipelines-provider"
)

func TestResourceDynamicPipelinesProvider_readProduction(t *testing.T) {
	rt := newScriptedTransport(t, dppJSONHandler(http.StatusOK, testDPPProductionARI, false))
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}

	assertDPPState(t, d, testDPPProductionARI, false)
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_readDevelopment(t *testing.T) {
	rt := newScriptedTransport(t, dppJSONHandler(http.StatusOK, testDPPDevelopmentARI, false))
	d := testDPPData(t, testDPPDevelopmentARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}

	assertDPPState(t, d, testDPPDevelopmentARI, false)
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_readNotConfigured(t *testing.T) {
	rt := newScriptedTransport(t, dppNotConfiguredTopLevelHandler())
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Fatalf("expected empty ID after provider-not-configured, got %q", d.Id())
	}
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_readNotConfiguredNested(t *testing.T) {
	rt := newScriptedTransport(t, dppNotConfiguredNestedHandler())
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != "" {
		t.Fatalf("expected empty ID after nested provider-not-configured, got %q", d.Id())
	}
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_readGeneric404(t *testing.T) {
	rt := newScriptedTransport(t, dppErrorHandler(http.StatusNotFound, `{"error":{"message":"Repository not found"}}`))
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if !diags.HasError() {
		t.Fatal("expected error for generic 404")
	}
	if d.Id() == "" {
		t.Fatal("generic 404 must not clear ID")
	}
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_read401(t *testing.T) {
	rt := newScriptedTransport(t, dppErrorHandler(http.StatusUnauthorized, `{"error":{"message":"Unauthorized"}}`))
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if !diags.HasError() {
		t.Fatal("expected error for 401")
	}
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_readMalformedJSON(t *testing.T) {
	rt := newScriptedTransport(t, dppRawHandler(http.StatusOK, `{"appAri":`))
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderRead(context.Background(), d, testDPPMeta(rt))
	if !diags.HasError() {
		t.Fatal("expected error for malformed JSON")
	}
	assertMethods(t, rt.calls, "GET")
}

func TestResourceDynamicPipelinesProvider_createSameARI(t *testing.T) {
	rt := newScriptedTransport(t, dppJSONHandler(http.StatusOK, testDPPProductionARI, false))
	d := testDPPData(t, testDPPProductionARI)

	diags := resourceDynamicPipelinesProviderCreate(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != testDPPID {
		t.Fatalf("expected ID %q, got %q", testDPPID, d.Id())
	}
	assertDPPState(t, d, testDPPProductionARI, false)
	assertMethods(t, rt.calls, "GET")
	assertNoPOST(t, rt.calls)
}

func TestResourceDynamicPipelinesProvider_createDifferentARI(t *testing.T) {
	rt := newScriptedTransport(t, dppJSONHandler(http.StatusOK, testDPPDevelopmentARI, false))
	d := testDPPData(t, testDPPProductionARI)

	diags := resourceDynamicPipelinesProviderCreate(context.Background(), d, testDPPMeta(rt))
	if !diags.HasError() {
		t.Fatal("expected error when remote ARI differs")
	}
	if d.Id() != "" {
		t.Fatalf("expected empty ID on conflict, got %q", d.Id())
	}
	assertMethods(t, rt.calls, "GET")
	assertNoPOST(t, rt.calls)
}

func TestResourceDynamicPipelinesProvider_createNone(t *testing.T) {
	rt := newScriptedTransport(t,
		dppNotConfiguredTopLevelHandler(),
		dppPostOKHandler(),
		dppJSONHandler(http.StatusOK, testDPPProductionARI, false),
	)
	d := testDPPData(t, testDPPProductionARI)

	diags := resourceDynamicPipelinesProviderCreate(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if d.Id() != testDPPID {
		t.Fatalf("expected ID %q, got %q", testDPPID, d.Id())
	}
	assertDPPState(t, d, testDPPProductionARI, false)
	assertMethods(t, rt.calls, "GET", "POST", "GET")
	if rt.calls[1].Body != `{"appAri":"`+testDPPProductionARI+`"}` {
		t.Fatalf("unexpected POST body: %s", rt.calls[1].Body)
	}
}

func TestResourceDynamicPipelinesProvider_updateSame(t *testing.T) {
	rt := newScriptedTransport(t, dppJSONHandler(http.StatusOK, testDPPProductionARI, false))
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderUpdate(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	assertDPPState(t, d, testDPPProductionARI, false)
	assertMethods(t, rt.calls, "GET")
	assertNoPOST(t, rt.calls)
}

func TestResourceDynamicPipelinesProvider_updateChanged(t *testing.T) {
	rt := newScriptedTransport(t,
		dppJSONHandler(http.StatusOK, testDPPDevelopmentARI, false),
		dppPostOKHandler(),
		dppJSONHandler(http.StatusOK, testDPPProductionARI, false),
	)
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderUpdate(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	assertDPPState(t, d, testDPPProductionARI, false)
	assertMethods(t, rt.calls, "GET", "POST", "GET")
	if rt.calls[1].Body != `{"appAri":"`+testDPPProductionARI+`"}` {
		t.Fatalf("unexpected POST body: %s", rt.calls[1].Body)
	}
}

func TestResourceDynamicPipelinesProvider_verificationMismatch(t *testing.T) {
	rt := newScriptedTransport(t,
		dppNotConfiguredTopLevelHandler(),
		dppPostOKHandler(),
		dppJSONHandler(http.StatusOK, testDPPDevelopmentARI, false),
	)
	d := testDPPData(t, testDPPProductionARI)

	diags := resourceDynamicPipelinesProviderCreate(context.Background(), d, testDPPMeta(rt))
	if !diags.HasError() {
		t.Fatal("expected verification mismatch error")
	}
	if !strings.Contains(diags[0].Summary, "verification failed") {
		t.Fatalf("expected verification error, got: %s", diags[0].Summary)
	}
	assertMethods(t, rt.calls, "GET", "POST", "GET")
}

func TestResourceDynamicPipelinesProvider_deleteZeroMutations(t *testing.T) {
	rt := newScriptedTransport(t)
	d := testDPPData(t, testDPPProductionARI)
	d.SetId(testDPPID)

	diags := resourceDynamicPipelinesProviderDelete(context.Background(), d, testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	if len(rt.calls) != 0 {
		t.Fatalf("delete must not perform HTTP calls, got %#v", rt.calls)
	}
}

func TestResourceDynamicPipelinesProvider_importWorkspaceRepository(t *testing.T) {
	r := resourceDynamicPipelinesProvider()
	if r.Importer == nil || r.Importer.StateContext == nil {
		t.Fatal("expected importer")
	}

	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{})
	d.SetId(testDPPID)

	imported, err := r.Importer.StateContext(context.Background(), d, nil)
	if err != nil {
		t.Fatalf("import failed: %s", err)
	}
	if len(imported) != 1 {
		t.Fatalf("expected 1 imported resource, got %d", len(imported))
	}
	if imported[0].Id() != testDPPID {
		t.Fatalf("expected imported ID %q, got %q", testDPPID, imported[0].Id())
	}

	rt := newScriptedTransport(t, dppJSONHandler(http.StatusOK, testDPPProductionARI, false))
	diags := resourceDynamicPipelinesProviderRead(context.Background(), imported[0], testDPPMeta(rt))
	if diags.HasError() {
		t.Fatalf("unexpected error: %s", diags[0].Summary)
	}
	assertDPPState(t, imported[0], testDPPProductionARI, false)
	assertMethods(t, rt.calls, "GET")
}

type httpCall struct {
	Method string
	Path   string
	Body   string
}

type scriptedTransport struct {
	t        *testing.T
	calls    []httpCall
	handlers []func(*http.Request) *http.Response
}

func (s *scriptedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.t.Helper()

	if req.URL.Host != "api.bitbucket.org" {
		s.t.Fatalf("unexpected host %q; tests must not reach a real network", req.URL.Host)
	}

	var body string
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			s.t.Fatalf("reading request body: %s", err)
		}
		body = string(b)
	}

	s.calls = append(s.calls, httpCall{
		Method: req.Method,
		Path:   req.URL.Path,
		Body:   body,
	})

	idx := len(s.calls) - 1
	if idx >= len(s.handlers) {
		s.t.Fatalf("unexpected HTTP call %s %s", req.Method, req.URL.String())
	}

	if req.URL.Path != testDPPPath {
		s.t.Fatalf("unexpected path %q, want %q", req.URL.Path, testDPPPath)
	}

	return s.handlers[idx](req), nil
}

func newScriptedTransport(t *testing.T, handlers ...func(*http.Request) *http.Response) *scriptedTransport {
	t.Helper()
	return &scriptedTransport{t: t, handlers: handlers}
}

func testDPPMeta(rt *scriptedTransport) Clients {
	return Clients{
		httpClient: Client{
			HTTPClient: &http.Client{Transport: rt},
		},
	}
}

func testDPPData(t *testing.T, appARI string) *schema.ResourceData {
	t.Helper()
	return schema.TestResourceDataRaw(t, resourceDynamicPipelinesProvider().Schema, map[string]interface{}{
		"workspace":  testDPPWorkspace,
		"repository": testDPPRepository,
		"app_ari":    appARI,
	})
}

func dppJSONHandler(status int, appARI string, fallback bool) func(*http.Request) *http.Response {
	body := fmt.Sprintf(`{"appAri":%q,"fallbackEnabled":%t}`, appARI, fallback)
	return dppRawHandler(status, body)
}

func dppNotConfiguredTopLevelHandler() func(*http.Request) *http.Response {
	return dppErrorHandler(http.StatusNotFound, `{"key":"account-service.dynamic-pipelines.provider-not-configured","message":"Dynamic Pipelines provider is not configured"}`)
}

func dppNotConfiguredNestedHandler() func(*http.Request) *http.Response {
	return dppErrorHandler(http.StatusNotFound, `{"type":"error","error":{"key":"account-service.dynamic-pipelines.provider-not-configured","message":"provider not configured"}}`)
}

func dppErrorHandler(status int, body string) func(*http.Request) *http.Response {
	return dppRawHandler(status, body)
}

func dppPostOKHandler() func(*http.Request) *http.Response {
	return dppRawHandler(http.StatusOK, `{}`)
}

func dppRawHandler(status int, body string) func(*http.Request) *http.Response {
	return func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
			Request:    req,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
	}
}

func assertDPPState(t *testing.T, d *schema.ResourceData, appARI string, fallback bool) {
	t.Helper()
	if d.Get("workspace").(string) != testDPPWorkspace {
		t.Fatalf("workspace = %q", d.Get("workspace"))
	}
	if d.Get("repository").(string) != testDPPRepository {
		t.Fatalf("repository = %q", d.Get("repository"))
	}
	if d.Get("app_ari").(string) != appARI {
		t.Fatalf("app_ari = %q, want %q", d.Get("app_ari"), appARI)
	}
	if d.Get("fallback_enabled").(bool) != fallback {
		t.Fatalf("fallback_enabled = %v, want %v", d.Get("fallback_enabled"), fallback)
	}
}

func assertMethods(t *testing.T, calls []httpCall, want ...string) {
	t.Helper()
	if len(calls) != len(want) {
		t.Fatalf("HTTP calls = %d (%#v), want %v", len(calls), calls, want)
	}
	for i, method := range want {
		if calls[i].Method != method {
			t.Fatalf("call %d method = %q, want %q", i, calls[i].Method, method)
		}
	}
}

func assertNoPOST(t *testing.T, calls []httpCall) {
	t.Helper()
	for _, call := range calls {
		switch call.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			t.Fatalf("unexpected mutating HTTP call: %#v", call)
		}
	}
}
