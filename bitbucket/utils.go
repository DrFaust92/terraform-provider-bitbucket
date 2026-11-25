package bitbucket

import (
	"bytes"
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Taken from https://github.com/hashicorp/terraform-provider-aws/blob/187f1659a4fef11ac314567273b5470afe6b662f/internal/service/iam/acc_test.go
// RandSSHKeyPair generates a public and private SSH key pair. The public key is
// returned in OpenSSH format, and the private key is PEM encoded.
// Copied from github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest,
//
//	with the addition of the key size
func RandSSHKeyPairSize(keySize int, comment string) (string, string, error) {
	privateKey, privateKeyPEM, err := genPrivateKey(keySize)
	if err != nil {
		return "", "", err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	keyMaterial := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey)))
	return fmt.Sprintf("%s %s", keyMaterial, comment), privateKeyPEM, nil
}

func RandPlainSSHKeyPairSize(keySize int) (string, error) {
	privateKey, _, err := RandSSHKeyPairSize(keySize, "")
	if err != nil {
		return "", err
	}

	return strings.Split(privateKey, " ")[1], nil
}

func genPrivateKey(keySize int) (*rsa.PrivateKey, string, error) {
	privateKey, err := rsa.GenerateKey(crand.Reader, keySize)
	if err != nil {
		return nil, "", err
	}

	privateKeyPEM, err := pemEncode(x509.MarshalPKCS1PrivateKey(privateKey), "RSA PRIVATE KEY")
	if err != nil {
		return nil, "", err
	}

	return privateKey, privateKeyPEM, nil
}

func pemEncode(b []byte, block string) (string, error) {
	var buf bytes.Buffer
	pb := &pem.Block{Type: block, Bytes: b}
	if err := pem.Encode(&buf, pb); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// formatTime formats t in RFC 3339, or returns "" for the zero time.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
