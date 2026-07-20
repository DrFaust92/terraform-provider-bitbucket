package bitbucket

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/armor"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// randGPGPublicKey generates a fresh ASCII-armored GPG public key so each
// acceptance run uses a unique fingerprint (Bitbucket rejects duplicates).
func randGPGPublicKey(name, email string) (string, error) {
	entity, err := openpgp.NewEntity(name, "terraform-acctest", email, nil)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	w, err := armor.Encode(&buf, openpgp.PublicKeyType, nil)
	if err != nil {
		return "", err
	}
	if err := entity.Serialize(w); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func TestAccBitbucketGpgKey_basic(t *testing.T) {
	resourceName := "bitbucket_gpg_key.test"

	userEmail := os.Getenv("BITBUCKET_USERNAME")
	publicKey, err := randGPGPublicKey("tf-acctest", userEmail)
	if err != nil {
		t.Fatalf("error generating GPG key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketGpgKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketGpgKeyConfig(publicKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketGpgKeyExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "user", "data.bitbucket_current_user.test", "uuid"),
					resource.TestCheckResourceAttr(resourceName, "key", publicKey),
					resource.TestCheckResourceAttrSet(resourceName, "fingerprint"),
					resource.TestCheckResourceAttrSet(resourceName, "key_id"),
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

func TestAccBitbucketGpgKey_name(t *testing.T) {
	resourceName := "bitbucket_gpg_key.test"

	rName := acctest.RandomWithPrefix("tf-test")
	userEmail := os.Getenv("BITBUCKET_USERNAME")
	publicKey, err := randGPGPublicKey("tf-acctest", userEmail)
	if err != nil {
		t.Fatalf("error generating GPG key: %s", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketGpgKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketGpgKeyNameConfig(publicKey, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketGpgKeyExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "user", "data.bitbucket_current_user.test", "uuid"),
					resource.TestCheckResourceAttr(resourceName, "key", publicKey),
					resource.TestCheckResourceAttr(resourceName, "name", rName),
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

func testAccCheckBitbucketGpgKeyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(Clients).genClient
	gpgApi := client.ApiClient.GPGApi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_gpg_key" {
			continue
		}

		user, fingerprint, err := gpgKeyId(rs.Primary.ID)
		if err != nil {
			return err
		}

		_, res, _ := gpgApi.UsersSelectedUserGpgKeysFingerprintGet(client.AuthContext, fingerprint, user)
		if res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("GPG Key still exists")
		}
	}
	return nil
}

func testAccCheckBitbucketGpgKeyExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No GPG Key ID is set")
		}
		return nil
	}
}

func testAccBitbucketGpgKeyConfig(pubkey string) string {
	return fmt.Sprintf(`
data "bitbucket_current_user" "test" {}

resource "bitbucket_gpg_key" "test" {
  user = data.bitbucket_current_user.test.uuid
  key  = %[1]q
}
`, pubkey)
}

func testAccBitbucketGpgKeyNameConfig(pubkey, name string) string {
	return fmt.Sprintf(`
data "bitbucket_current_user" "test" {}

resource "bitbucket_gpg_key" "test" {
  user = data.bitbucket_current_user.test.uuid
  key  = %[1]q
  name = %[2]q
}
`, pubkey, name)
}
