package bitbucket

import (
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
)

func TestSuppressSSHKeyCommentDiff(t *testing.T) {
	cases := []struct {
		old, new string
		want     bool
	}{
		{"ssh-ed25519 AAAAC3Nza", "ssh-ed25519 AAAAC3Nza user@host", true},
		{"ssh-ed25519 AAAAC3Nza old-comment", "ssh-ed25519 AAAAC3Nza new-comment", true},
		{"ssh-ed25519 AAAAC3Nza", "ssh-ed25519 AAAAC3Nzb", false},
		{"ssh-rsa AAAAC3Nza", "ssh-ed25519 AAAAC3Nza", false},
		{"", "ssh-ed25519 AAAAC3Nza", false},
	}
	for _, tc := range cases {
		if got := suppressSSHKeyCommentDiff("key", tc.old, tc.new, nil); got != tc.want {
			t.Errorf("suppressSSHKeyCommentDiff(%q, %q) = %v, want %v", tc.old, tc.new, got, tc.want)
		}
	}
}

func TestGPGKeyFingerprint(t *testing.T) {
	armored, err := randGPGPublicKey("tf unit", "unit@example.com")
	if err != nil {
		t.Fatal(err)
	}
	keys, err := openpgp.ReadArmoredKeyRing(strings.NewReader(armored))
	if err != nil {
		t.Fatal(err)
	}
	want := keys[0].PrimaryKey.Fingerprint

	got := gpgKeyFingerprint(armored)
	if len(got) != 2*len(want) {
		t.Fatalf("got fingerprint %q, want %d hex chars", got, 2*len(want))
	}
	if gpgKeyFingerprint("not a key") != "" {
		t.Fatal("expected empty fingerprint for invalid input")
	}

	other, err := randGPGPublicKey("tf other", "other@example.com")
	if err != nil {
		t.Fatal(err)
	}
	d := resourceGpgKey().TestResourceData()
	if err := d.Set("fingerprint", strings.ToUpper(got)); err != nil {
		t.Fatal(err)
	}
	if !suppressSameGPGKey("key", other, armored, d) {
		t.Error("expected a re-armored copy of the same key to be suppressed")
	}
	if suppressSameGPGKey("key", armored, other, d) {
		t.Error("expected a different key not to be suppressed")
	}
}
