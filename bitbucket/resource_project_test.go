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

func TestAccBitbucketProject_basic(t *testing.T) {
	resourceName := "bitbucket_project.test"
	testTeam := os.Getenv("BITBUCKET_TEAM")
	rName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketProjectConfig(testTeam, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "has_publicly_visible_repos", "false"),
					resource.TestCheckResourceAttr(resourceName, "key", "AAAAAA"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "owner", testTeam),
					resource.TestCheckResourceAttr(resourceName, "description", ""),
					resource.TestCheckResourceAttr(resourceName, "is_private", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "uuid"),
					resource.TestCheckResourceAttr(resourceName, "link.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "link.0.avatar.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "link.0.avatar.0.href"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccBitbucketProjectDescConfig(testTeam, rName, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "has_publicly_visible_repos", "false"),
					resource.TestCheckResourceAttr(resourceName, "key", "AAAAAA"),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
					resource.TestCheckResourceAttr(resourceName, "owner", testTeam),
					resource.TestCheckResourceAttr(resourceName, "description", rName),
					resource.TestCheckResourceAttr(resourceName, "is_private", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "uuid"),
					resource.TestCheckResourceAttr(resourceName, "link.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "link.0.avatar.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "link.0.avatar.0.href"),
				),
			},
		},
	})
}

func TestAccBitbucketProject_avatar(t *testing.T) {
	resourceName := "bitbucket_project.test"
	testTeam := os.Getenv("BITBUCKET_TEAM")
	rName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketProjectDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketProjectAvatarConfig(testTeam, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketProjectExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "link.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "link.0.avatar.#", "1"),
					resource.TestCheckResourceAttrSet(resourceName, "link.0.avatar.0.href"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccBitbucketProjectConfig(team, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_project" "test" {
  owner = %[1]q
  name  = %[2]q
  key   = "AAAAAA"
}
`, team, rName)
}

func testAccBitbucketProjectDescConfig(team, rName, desc string) string {
	return fmt.Sprintf(`
resource "bitbucket_project" "test" {
  owner       = %[1]q
  name        = %[2]q
  key         = "AAAAAA"
  description = %[3]q
}
`, team, rName, desc)
}

func testAccBitbucketProjectAvatarConfig(team, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_project" "test" {
  owner = %[1]q
  name  = %[2]q
  key   = "BBBBB"

  link {
    avatar {
      href = "https://d301sr5gafysq2.cloudfront.net/dfb18959be9c/img/repo-avatars/python.png"
	}
  }
}
`, team, rName)
}

func testAccCheckBitbucketProjectDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(Clients).genClient
	projectApi := client.ApiClient.ProjectsApi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_project" {
			continue
		}

		_, res, err := projectApi.WorkspacesWorkspaceProjectsProjectKeyGet(client.AuthContext,
			rs.Primary.Attributes["key"], rs.Primary.Attributes["owner"])

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Project still exists")
		}
	}
	return nil
}

func testAccCheckBitbucketProjectExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No project ID is set")
		}
		return nil
	}
}
