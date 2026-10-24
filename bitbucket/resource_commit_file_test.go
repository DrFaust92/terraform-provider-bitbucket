package bitbucket

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func testAccBitbucketCommitFileConfig(owner, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_commit_file" "test" {
	filename       = "README.md"
	content        = "abc"
	repo_slug      = bitbucket_repository.test.name
	workspace      = bitbucket_repository.test.owner
	commit_author  = "Unit test <unit@test.local>"
	branch         = "main"
	commit_message = "test"
  }
`, owner, rName)
}

func TestAccBitbucketCommitFile_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	owner := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_commit_file.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDefaultReviewersDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketCommitFileConfig(owner, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDefaultReviewersExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_commit_file.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "content", "abc"),
					resource.TestCheckResourceAttr(resourceName, "branch", "main"),
				),
			},
		},
	})
}

func TestAccBitbucketCommitFile_parallelSameBranch(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	owner := os.Getenv("BITBUCKET_TEAM")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_commit_file" "test" {
  count          = 5
  filename       = "file-${count.index}.txt"
  content        = "content ${count.index}"
  repo_slug      = bitbucket_repository.test.name
  workspace      = bitbucket_repository.test.owner
  commit_author  = "Unit test <unit@test.local>"
  branch         = "main"
  commit_message = "add file ${count.index}"
}
`, owner, rName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("bitbucket_commit_file.test.0", "commit_sha"),
					resource.TestCheckResourceAttrSet("bitbucket_commit_file.test.4", "commit_sha"),
				),
			},
		},
	})
}
