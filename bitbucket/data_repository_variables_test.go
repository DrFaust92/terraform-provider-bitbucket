package bitbucket

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceRepositoryVariables_basic(t *testing.T) {
	dataSourceName := "data.bitbucket_repository_variables.test"
	resourceName := "bitbucket_repository_variable.test"

	rName := acctest.RandomWithPrefix("tf-test")
	workspace := os.Getenv("BITBUCKET_TEAM")

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketRepositoryVariablesConfig(workspace, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "variables.#", "1"),
					resource.TestCheckResourceAttrPair(dataSourceName, "variables.0.uuid", resourceName, "uuid"),
					resource.TestCheckResourceAttr(dataSourceName, "variables.0.key", "test"),
					resource.TestCheckResourceAttr(dataSourceName, "variables.0.value", "test-val"),
					resource.TestCheckResourceAttr(dataSourceName, "variables.0.secured", "false"),
				),
			},
		},
	})
}

func testAccBitbucketRepositoryVariablesConfig(workspace, rName string) string {
	return testAccBitbucketRepositoryVariableConfig(workspace, rName, "test-val") + fmt.Sprintf(`
data "bitbucket_repository_variables" "test" {
  workspace  = %[1]q
  repository = bitbucket_repository.test.slug

  depends_on = [bitbucket_repository_variable.test]
}
`, workspace)
}

func TestDataRepositoryVariablesReadFollowsPages(t *testing.T) {
	var calls int32
	server := repositoryVariablesServer(t, 12, &calls)
	defer server.Close()

	d := dataRepositoryVariables().TestResourceData()
	d.Set("workspace", "workspace")
	d.Set("repository", "repo")

	if diags := dataReadRepositoryVariables(context.Background(), d, testRepositoryVariableClients(server.URL)); diags.HasError() {
		t.Fatalf("read failed: %v", diags)
	}

	if d.Id() != "workspace/repo" {
		t.Errorf("id = %q, want workspace/repo", d.Id())
	}
	if n := d.Get("variables.#").(int); n != 12 {
		t.Fatalf("got %d variables, want all 12 across both pages", n)
	}
	// Sorted by key: KEY_1, KEY_10, KEY_11, KEY_12, KEY_2, ...
	if got := d.Get("variables.3.key").(string); got != "KEY_12" {
		t.Errorf("variables.3.key = %q, want KEY_12", got)
	}
	if got := d.Get("variables.3.uuid").(string); got != "{var-12}" {
		t.Errorf("variables.3.uuid = %q, want {var-12}", got)
	}
	if !d.Get("variables.3.secured").(bool) {
		t.Errorf("variables.3.secured = false, want true")
	}
}
