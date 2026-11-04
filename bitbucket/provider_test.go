package bitbucket

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider

var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"bitbucket": func() (*schema.Provider, error) { return testAccProvider, nil },
}

func init() {
	testAccProvider = Provider()
	testAccProviders = map[string]*schema.Provider{
		"bitbucket": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ = Provider()
}

func testAccPreCheck(t *testing.T) {

	// Allow either bitbucket u/p or oauth creds for testing
	user_v := os.Getenv("BITBUCKET_USERNAME")
	pass_v := os.Getenv("BITBUCKET_PASSWORD")
	oauth_token_v := os.Getenv("BITBUCKET_OAUTH_TOKEN")
	oauth_client_v := os.Getenv("BITBUCKET_OAUTH_CLIENT_ID")
	oauth_secret_v := os.Getenv("BITBUCKET_OAUTH_CLIENT_SECRET")

	if user_v == "" && oauth_token_v == "" && oauth_client_v == "" {
		t.Fatal("BITBUCKET_USERNAME or BITBUCKET_OAUTH_TOKEN or BITBUCKET_OAUTH_CLIENT_ID must be set for acceptance tests")
	}

	if (pass_v == "" && user_v != "") || (pass_v != "" && (oauth_token_v != "" || oauth_secret_v != "")) {
		t.Fatal("BITBUCKET_PASSWORD must be set if using BITBUCKET_USERNAME for acceptance tests")
	}

	if (oauth_secret_v == "" && oauth_client_v != "") || (oauth_secret_v != "" && (oauth_token_v != "" || user_v != "")) {
		t.Fatal("BITBUCKET_OAUTH_CLIENT_SECRET must be set if using BITBUCKET_OAUTH_CLIENT_ID for acceptance tests")
	}

	if v := os.Getenv("BITBUCKET_TEAM"); v == "" {
		t.Fatal("BITBUCKET_TEAM must be set for acceptance tests")
	}
}

func testAccPreCheckPipeSchedule(t *testing.T) {
	if v := os.Getenv("BITBUCKET_PIPELINED_REPO"); v == "" {
		t.Fatal("BITBUCKET_PIPELINED_REPO must be set for acceptance tests")
	}
}

func testAccPreCheckRepo(t *testing.T) {
	if v := os.Getenv("BITBUCKET_REPO"); v == "" {
		t.Fatal("BITBUCKET_REPO must be set for repo and file datasource acceptance tests")
	}
}

func testAccPreCheckFileCommit(t *testing.T) {
	if v := os.Getenv("BITBUCKET_COMMIT"); v == "" {
		t.Fatal("BITBUCKET_COMMIT must be set for file datasource acceptance tests")
	}
	if v := os.Getenv("BITBUCKET_PATH"); v == "" {
		t.Fatal("BITBUCKET_PATH must be set for file datasource acceptance tests")
	}
}

func testAccPreCheckFilePath(t *testing.T) {
	if v := os.Getenv("BITBUCKET_PATH"); v == "" {
		t.Fatal("BITBUCKET_PATH must be set for file datasource acceptance tests")
	}
}

func testAccPreCheckProject(t *testing.T) {
	if v := os.Getenv("BITBUCKET_PROJECT"); v == "" {
		t.Fatal("BITBUCKET_PROJECT must be set for project datasource acceptance tests")
	}
}
