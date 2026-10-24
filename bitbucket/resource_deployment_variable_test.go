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

func TestAccBitbucketDeploymentVariable_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	owner := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_deployment_variable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDeploymentVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketDeploymentVariableConfig(owner, rName, "test", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeploymentVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "deployment", "bitbucket_deployment.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test"),
					resource.TestCheckResourceAttr(resourceName, "secured", "false"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccBitbucketDeploymentVariableImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccBitbucketDeploymentVariableConfig(owner, rName, "test-2", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeploymentVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "deployment", "bitbucket_deployment.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test-2"),
					resource.TestCheckResourceAttr(resourceName, "secured", "false"),
				),
			},
		},
	})
}

func TestAccBitbucketDeploymentVariable_manyVars(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	owner := os.Getenv("BITBUCKET_TEAM")
	// resourceName := "bitbucket_deployment_variable.test[0]"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDeploymentVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketDeploymentVariableManyConfig(owner, rName, "test", false),
			},
		},
	})
}

func TestAccBitbucketDeploymentVariable_secure(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	owner := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_deployment_variable.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketDeploymentVariableDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketDeploymentVariableConfig(owner, rName, "test", true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeploymentVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "deployment", "bitbucket_deployment.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test"),
					resource.TestCheckResourceAttr(resourceName, "secured", "true"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateIdFunc:       testAccBitbucketDeploymentVariableImportStateIdFunc(resourceName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"value"},
			},
			{
				Config: testAccBitbucketDeploymentVariableConfig(owner, rName, "test", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeploymentVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "deployment", "bitbucket_deployment.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test"),
					resource.TestCheckResourceAttr(resourceName, "secured", "false"),
				),
			},
			{
				Config: testAccBitbucketDeploymentVariableConfig(owner, rName, "test", true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketDeploymentVariableExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "deployment", "bitbucket_deployment.test", "id"),
					resource.TestCheckResourceAttr(resourceName, "key", "test"),
					resource.TestCheckResourceAttr(resourceName, "value", "test"),
					resource.TestCheckResourceAttr(resourceName, "secured", "true"),
				),
			},
		},
	})
}

func testAccCheckBitbucketDeploymentVariableDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(Clients).genClient
	pipeApi := client.ApiClient.PipelinesApi
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_deployment_variable" {
			continue
		}

		repository, deployment, err := deploymentId(rs.Primary.Attributes["deployment"])
		if err != nil {
			return err
		}
		workspace, repoSlug, err := deployVarId(repository)
		if err != nil {
			return err
		}

		_, res, err := pipeApi.GetDeploymentVariables(client.AuthContext, workspace, repoSlug, deployment, nil)

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("Deployment Variable still exists")
		}
	}
	return nil
}

func testAccCheckBitbucketDeploymentVariableExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("Not found %s", n)
		}

		return nil
	}
}

func testAccBitbucketDeploymentVariableConfig(owner, rName, val string, secure bool) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_deployment" "test" {
  name       = %[2]q
  stage      = "Test"
  repository = bitbucket_repository.test.id
}

resource "bitbucket_deployment_variable" "test" {
  key        = "test"
  value      = %[3]q
  deployment = bitbucket_deployment.test.id
  secured    = %[4]t
}
`, owner, rName, val, secure)
}

func testAccBitbucketDeploymentVariableManyConfig(owner, rName, val string, secure bool) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_deployment" "test" {
  name       = %[2]q
  stage      = "Test"
  repository = bitbucket_repository.test.id
}

resource "bitbucket_deployment_variable" "test" {
  count = 50

  key        = "test${count.index}"
  value      = %[3]q
  deployment = bitbucket_deployment.test.id
  secured    = %[4]t
}
`, owner, rName, val, secure)
}

func testAccBitbucketDeploymentVariableImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s", rs.Primary.Attributes["deployment"], rs.Primary.ID), nil
	}
}
