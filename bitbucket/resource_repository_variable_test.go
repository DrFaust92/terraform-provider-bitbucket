package bitbucket

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccBitbucketRepositoryVariable_basic(t *testing.T) {

	owner := os.Getenv("BITBUCKET_TEAM")
	rName := acctest.RandomWithPrefix("tf-test")
	resourceName := "bitbucket_repository_variable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketRepositoryVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketRepositoryVariableConfig(owner, rName, "test-val"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketRepositoryVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test-val"),
					resource.TestCheckResourceAttr(resourceName, "secured", "false"),
					resource.TestCheckResourceAttr(resourceName, "workspace", owner),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccBitbucketRepoVariableImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccBitbucketRepoVariableImportByKeyStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccBitbucketRepositoryVariableConfig(owner, rName, "test-val-2"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketRepositoryVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test-val-2"),
					resource.TestCheckResourceAttr(resourceName, "secured", "false"),
				),
			},
		},
	})
}

func testAccCheckBitbucketRepositoryVariableDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(Clients).genClient
	pipeApi := client.ApiClient.PipelinesApi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_repository_variable" {
			continue
		}

		workspace, repoSlug, err := repoVarId(rs.Primary.Attributes["repository"])
		if err != nil {
			return err
		}

		_, res, err := pipeApi.GetRepositoryPipelineVariable(client.AuthContext, workspace, repoSlug, rs.Primary.Attributes["uuid"])

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Repository Variable still exists")
		}
	}
	return nil
}

func testAccCheckBitbucketRepositoryVariableExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found %s", n)
		}

		return nil
	}
}

func testAccBitbucketRepositoryVariableConfig(team, rName, val string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_repository_variable" "test" {
  key        = "test"
  value      = %[3]q
  repository = bitbucket_repository.test.id
}
`, team, rName, val)
}

func testAccBitbucketRepoVariableImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["repository"], rs.Primary.ID, rs.Primary.Attributes["uuid"]), nil
	}
}

func testAccBitbucketRepoVariableImportByKeyStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}

		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["repository"], rs.Primary.ID), nil
	}
}
