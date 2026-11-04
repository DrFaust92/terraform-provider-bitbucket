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

func TestAccBitbucketDeployKey_basic(t *testing.T) {
	var deployKey SshKey
	resourceName := "bitbucket_deploy_key.test"

	rName := acctest.RandomWithPrefix("tf-test")
	owner := os.Getenv("BITBUCKET_TEAM")
	userEmail := os.Getenv("BITBUCKET_USERNAME")
	publicKey, _, err := RandSSHKeyPairSize(2048, userEmail)
	if err != nil {
		t.Fatalf("error generating random SSH key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDeployKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketDeployKeyConfig(owner, rName, publicKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeployKeyExists(resourceName, &deployKey),
					resource.TestCheckResourceAttrPair(resourceName, "workspace", "bitbucket_repository.test", "owner"),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"), resource.TestCheckResourceAttr(resourceName, "key", publicKey),
					resource.TestCheckResourceAttr(resourceName, "comment", userEmail),
					resource.TestCheckResourceAttrSet(resourceName, "key_id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
			{
				ResourceName:       resourceName,
				ImportState:        true,
				ImportStatePersist: true,
			},
			{
				Config:   testAccBitbucketDeployKeyConfig(owner, rName, publicKey),
				PlanOnly: true,
			},
		},
	})
}

func TestAccBitbucketDeployKey_label(t *testing.T) {
	var deployKey SshKey
	resourceName := "bitbucket_deploy_key.test"

	owner := os.Getenv("BITBUCKET_TEAM")
	rName := acctest.RandomWithPrefix("tf-test")
	userEmail := os.Getenv("BITBUCKET_USERNAME")
	publicKey, _, err := RandSSHKeyPairSize(2048, userEmail)
	if err != nil {
		t.Fatalf("error generating random SSH key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDeployKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketDeployKeyLabelConfig(owner, rName, publicKey, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeployKeyExists(resourceName, &deployKey),
					resource.TestCheckResourceAttrPair(resourceName, "workspace", "bitbucket_repository.test", "owner"),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "key", publicKey),
					resource.TestCheckResourceAttr(resourceName, "label", rName),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

func testAccCheckBitbucketDeployKeyDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(Clients).genClient
	deployApi := c.ApiClient.DeploymentsApi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_deploy_key" {
			continue
		}

		workspace, repo, keyId, err := deployKeyId(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, response, err := deployApi.RepositoriesWorkspaceRepoSlugDeployKeysKeyIdGet(c.AuthContext, keyId, repo, workspace)

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if response.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Deploy Key still exists")
		}

	}
	return nil
}

func testAccCheckBitbucketDeployKeyExists(n string, deployKey *SshKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No Deploy Key ID is set")
		}
		return nil
	}
}

func testAccBitbucketDeployKeyConfig(workspace, rName, pubkey string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_deploy_key" "test" {
  workspace  = %[1]q
  repository = bitbucket_repository.test.name
  key        = %[3]q
}
`, workspace, rName, pubkey)
}

func testAccBitbucketDeployKeyLabelConfig(workspace, rName, pubkey, label string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_deploy_key" "test" {
  workspace  = %[1]q
  repository = bitbucket_repository.test.name
  key        = %[3]q
  label      = %[4]q
}
`, workspace, rName, pubkey, label)
}
