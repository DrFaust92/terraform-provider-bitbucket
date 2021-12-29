package bitbucket

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	uuid "github.com/satori/go.uuid"
)

func TestAccBitbucketHook_basic(t *testing.T) {
	var hook Hook
	resourceName := "bitbucket_hook.test"
	testUser := os.Getenv("BITBUCKET_TEAM")
	rName := acctest.RandomWithPrefix("tf-test")

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketHookDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketHookConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketHookExists(resourceName, &hook),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test hook for terraform"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://httpbin.org"),
					resource.TestCheckResourceAttr(resourceName, "skip_cert_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "skip_cert_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "events.#", "1"),
				),
			},
			{
				Config: testAccBitbucketHookConfigUpdated(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketHookExists(resourceName, &hook),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test hook for terraform Updated"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://httpbin.org"),
					resource.TestCheckResourceAttr(resourceName, "skip_cert_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "skip_cert_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "events.#", "2"),
				),
			},
			{
				Config: testAccBitbucketHookConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketHookExists(resourceName, &hook),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "description", "Test hook for terraform"),
					resource.TestCheckResourceAttr(resourceName, "url", "https://httpbin.org"),
					resource.TestCheckResourceAttr(resourceName, "skip_cert_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "skip_cert_verification", "true"),
					resource.TestCheckResourceAttr(resourceName, "events.#", "1"),
				),
			},
		},
	})
}

func TestEncodesJsonCompletely(t *testing.T) {
	hook := &Hook{
		UUID:        uuid.NewV4().String(),
		URL:         "https://site.internal/",
		Description: "Test description",
		Active:      false,
		Events: []string{
			"pullrequests:updated",
		},
		SkipCertVerification: false,
	}

	payload, err := json.Marshal(hook)
	if err != nil {
		t.Logf("Failed to encode hook, %s\n", err)
		t.FailNow() // Can't continue test.
	}

	if !strings.Contains(string(payload), `"active":false`) {
		t.Error("Did not render active.")
	}

	if !strings.Contains(string(payload), `"skip_cert_verification":false`) {
		t.Error("Did not render skip_cert_verification.")
	}
}

func testAccCheckBitbucketHookDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_hook" {
			continue
		}

		response, err := client.Get(fmt.Sprintf("2.0/repositories/%s/%s/hooks/%s", rs.Primary.Attributes["owner"], rs.Primary.Attributes["repository"], url.PathEscape(rs.Primary.Attributes["uuid"])))

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if response.StatusCode != 404 {
			return fmt.Errorf("Hook still exists")
		}

	}
	return nil
}

func testAccCheckBitbucketHookExists(n string, hook *Hook) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Hook ID is set")
		}
		return nil
	}
}

func testAccBitbucketHookConfig(testUser, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_hook" "test" {
  owner                  = %[1]q
  repository             = bitbucket_repository.test.name
  description            = "Test hook for terraform"
  url                    = "https://httpbin.org"
  skip_cert_verification = true

  events = [
  	"repo:push",
  ]
}
`, testUser, rName)
}

func testAccBitbucketHookConfigUpdated(testUser, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_hook" "test" {
  owner                  = %[1]q
  repository             = bitbucket_repository.test.name
  description            = "Test hook for terraform Updated"
  url                    = "https://httpbin.org"
  skip_cert_verification = true

  events = [
  	"repo:push",
    "repo:fork",
  ]
}
`, testUser, rName)
}
