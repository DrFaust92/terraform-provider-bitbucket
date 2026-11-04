package bitbucket

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDeployments_basic(t *testing.T) {
	dataSourceName := "data.bitbucket_deployments.test"
	resourceName := "bitbucket_deployment.test"

	rName := acctest.RandomWithPrefix("tf-test")

	workspace := os.Getenv("BITBUCKET_TEAM")
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketDeploymentsConfig(workspace, rName, rName),
			},
			{
				Config: testAccBitbucketDeploymentsConfig(workspace, rName, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "workspace", workspace),
					resource.TestCheckResourceAttr(dataSourceName, "uuids.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "names.#", "1"),
					resource.TestCheckResourceAttrPair(dataSourceName, "uuids.0", resourceName, "uuid"),
					resource.TestCheckResourceAttrPair(dataSourceName, "names.0", resourceName, "name"),
				),
			},
		},
	})
}

func testAccBitbucketDeploymentsConfig(workspace, repoName, deployName string) string {
	return testAccBitbucketDeployment(workspace, repoName, deployName) + fmt.Sprintf(`
data "bitbucket_deployments" "test" {
  workspace  = %[1]q
  repository = bitbucket_repository.test.slug

  depends_on = [bitbucket_deployment.test]
}
`, workspace)
}
