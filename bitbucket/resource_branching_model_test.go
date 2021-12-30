package bitbucket

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccBitbucketBranchingModel_basic(t *testing.T) {
	var branchRestriction BranchingModel
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branching_model.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketBranchingModelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchingModelConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchingModelExists(resourceName, &branchRestriction),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "development.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "development.0.use_mainbranch", "true"),
				),
			},
		},
	})
}

func testAccBitbucketBranchingModelConfig(testUser, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branching_model" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name

  development {
    use_mainbranch = true
  }
}
`, testUser, rName)
}

func testAccCheckBitbucketBranchingModelDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_branching_model" {
			continue
		}
		response, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/branching-model", rs.Primary.Attributes["owner"], rs.Primary.Attributes["repository"]))

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if response.StatusCode != 404 {
			return fmt.Errorf("Branching Model still exists")
		}
	}

	return nil
}

func testAccCheckBitbucketBranchingModelExists(n string, branchRestriction *BranchingModel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No BranchingModel ID is set")
		}
		return nil
	}
}
