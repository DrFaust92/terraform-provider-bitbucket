package bitbucket

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccBitbucketBranchRestriction_basic(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "kind", "force"),
					resource.TestCheckResourceAttr(resourceName, "pattern", "master"),
					resource.TestCheckResourceAttr(resourceName, "branch_match_kind", "glob"),
				),
			},
			{
				// Verify a plan immediately after apply is empty (no perpetual diff).
				Config:   testAccBitbucketBranchRestrictionConfig(testUser, rName),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBitbucketBranchRestriction_model(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionModelConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttrPair(resourceName, "repository", "bitbucket_repository.test", "name"),
					resource.TestCheckResourceAttr(resourceName, "kind", "force"),
					resource.TestCheckResourceAttr(resourceName, "pattern", ""),
					resource.TestCheckResourceAttr(resourceName, "branch_match_kind", "branching_model"),
					resource.TestCheckResourceAttr(resourceName, "branch_type", "production"),
				),
			},
			{
				// Verify a plan immediately after apply is empty (no perpetual diff).
				Config:   testAccBitbucketBranchRestrictionModelConfig(testUser, rName),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBitbucketBranchRestriction_users(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders, //nolint:staticcheck // pre-existing repo-wide pattern; ProviderFactories migration is out of scope
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionUsersConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "users.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "users.*", "data.bitbucket_current_user.test", "uuid"),
				),
			},
			{
				// Verify the UUID round-trip is stable: a plan immediately after
				// apply must be empty (no perpetual diff).
				Config:   testAccBitbucketBranchRestrictionUsersConfig(testUser, rName),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccBitbucketBranchRestriction_groups(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders, //nolint:staticcheck // pre-existing repo-wide pattern; ProviderFactories migration is out of scope
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionGroupsConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "groups.*.slug", "bitbucket_group.test", "slug"),
					// Confirmed: the 2.0 API returns owner.uuid and workspace.uuid in the
					// group object on both personal (davidlukac07) and org (sycle-corp)
					// workspaces. groups.owner round-trips as the workspace UUID.
					resource.TestCheckTypeSetElemAttrPair(resourceName, "groups.*.owner", "data.bitbucket_workspace.test", "id"),
				),
			},
			{
				// Verify the UUID round-trip is stable: a plan immediately after
				// apply must be empty (no perpetual diff).
				Config:   testAccBitbucketBranchRestrictionGroupsConfig(testUser, rName),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccBitbucketBranchRestriction_groupsExisting verifies the UUID round-trip
// for groups.owner without requiring group creation (which needs account:write,
// unavailable on modern Atlassian API tokens).
//
// It uses a pre-existing workspace group identified by BITBUCKET_EXISTING_GROUP_SLUG
// and grants it write access to the test repo before adding it to a branch
// restriction. It then verifies that:
//   - groups.owner in state matches the workspace UUID (not the slug)
//   - a plan immediately after apply is empty (no perpetual diff)
//   - ImportStateVerify passes
//
// For davidlukac07 use: BITBUCKET_EXISTING_GROUP_SLUG=da52073d-f30b-49e1-907a-be56bb5c0f5b
func TestAccBitbucketBranchRestriction_groupsExisting(t *testing.T) {
	groupSlug := os.Getenv("BITBUCKET_EXISTING_GROUP_SLUG")
	if groupSlug == "" {
		t.Skip("BITBUCKET_EXISTING_GROUP_SLUG must be set for this test")
	}

	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders, //nolint:staticcheck // pre-existing repo-wide pattern; ProviderFactories migration is out of scope
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionGroupsExistingConfig(testUser, rName, groupSlug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "groups.0.slug", groupSlug),
					// The 2.0 API returns owner.uuid for the group owner.
					// groupOwnerIdentifier picks Workspace.Uuid (or Owner.Uuid) so
					// state should contain the workspace UUID, not the slug.
					resource.TestCheckTypeSetElemAttrPair(resourceName, "groups.*.owner", "data.bitbucket_workspace.test", "id"),
				),
			},
			{
				// UUID round-trip: plan immediately after apply must be empty.
				Config:   testAccBitbucketBranchRestrictionGroupsExistingConfig(testUser, rName, groupSlug),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccBitbucketBranchRestriction_legacyUsername proves the documented
// perpetual-diff behavior end-to-end: Bitbucket's API no longer returns
// usernames, so a config still pinned to one never converges after apply.
// Gated on BITBUCKET_LEGACY_USERNAME (an existing, legacy-resolvable
// account) since this fixture is optional and not required by any other
// acceptance test.
func TestAccBitbucketBranchRestriction_legacyUsername(t *testing.T) {
	legacyUsername := os.Getenv("BITBUCKET_LEGACY_USERNAME")
	if legacyUsername == "" {
		t.Skip("BITBUCKET_LEGACY_USERNAME must be set for this test")
	}

	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders, //nolint:staticcheck // pre-existing repo-wide pattern; ProviderFactories migration is out of scope
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				// After apply, Read gets back a UUID (the API no longer returns
				// usernames), so the SDK's post-apply convergence check sees a
				// non-empty plan.  ExpectNonEmptyPlan here lets step 1 pass so
				// we can reach the explicit PlanOnly assertion in step 2.
				Config:             testAccBitbucketBranchRestrictionLegacyUsernameConfig(testUser, rName, legacyUsername),
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "users.#", "1"),
				),
			},
			{
				Config:             testAccBitbucketBranchRestrictionLegacyUsernameConfig(testUser, rName, legacyUsername),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestCreateBranchRestriction(t *testing.T) {
	raw := map[string]interface{}{
		"owner":      "myteam",
		"repository": "my-repo",
		"kind":       "push",
		"pattern":    "master",
		"value":      2,
		"users": []interface{}{
			"{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}",
			"legacy-username",
		},
		"groups": []interface{}{
			map[string]interface{}{
				"owner": "{deadbeef-dead-beef-dead-beefdeadbeef}",
				"slug":  "my-group",
			},
			map[string]interface{}{
				"owner": "legacy-group-owner",
				"slug":  "other-group",
			},
		},
	}

	d := schema.TestResourceDataRaw(t, resourceBranchRestriction().Schema, raw)
	got := createBranchRestriction(d)

	if got.Kind != "push" {
		t.Errorf("Kind = %q, want %q", got.Kind, "push")
	}
	if got.Pattern != "master" {
		t.Errorf("Pattern = %q, want %q", got.Pattern, "master")
	}
	if got.Value != 2 {
		t.Errorf("Value = %d, want %d", got.Value, 2)
	}

	wantUsers := []bitbucket.Account{
		{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
		{Username: "legacy-username"},
	}
	if !sameAccountSet(got.Users, wantUsers) {
		t.Errorf("Users = %+v, want (any order) %+v", got.Users, wantUsers)
	}

	wantGroups := map[string]bitbucket.Account{
		"my-group":    {Uuid: "{deadbeef-dead-beef-dead-beefdeadbeef}"},
		"other-group": {Username: "legacy-group-owner"},
	}
	if len(got.Groups) != len(wantGroups) {
		t.Fatalf("Groups len = %d, want %d", len(got.Groups), len(wantGroups))
	}
	for _, group := range got.Groups {
		wantOwner, ok := wantGroups[group.Slug]
		if !ok {
			t.Errorf("unexpected group slug %q", group.Slug)
			continue
		}
		if group.Owner == nil || !reflect.DeepEqual(*group.Owner, wantOwner) {
			t.Errorf("group %q owner = %+v, want %+v", group.Slug, group.Owner, wantOwner)
		}
	}
}

// sameAccountSet reports whether got and want contain the same bitbucket.Account
// values, ignoring order — createBranchRestriction builds users from a
// schema.Set, whose iteration order is not guaranteed.
func sameAccountSet(got, want []bitbucket.Account) bool {
	if len(got) != len(want) {
		return false
	}

	remaining := make([]bitbucket.Account, len(want))
	copy(remaining, want)

	for _, g := range got {
		found := false
		for i, w := range remaining {
			if reflect.DeepEqual(g, w) {
				remaining = append(remaining[:i], remaining[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func TestExpandBranchRestrictionUser(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bitbucket.Account
	}{
		{"empty string treated as username", "", bitbucket.Account{Username: ""}},
		{"username", "my-bitbucket-username", bitbucket.Account{Username: "my-bitbucket-username"}},
		{"uuid", "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", bitbucket.Account{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"}},
		{"uuid without braces treated as username", "c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee", bitbucket.Account{Username: "c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee"}},
		{"uuid mixed case", "{C0FFEE00-c0ff-EEC0-ffee-c0ffeeC0ffee}", bitbucket.Account{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := expandBranchRestrictionUser(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("expandBranchRestrictionUser(%q) = %+v, want %+v", tc.input, got, tc.want)
			}
		})
	}
}

func TestBranchRestrictionUserSetFunc(t *testing.T) {
	setFunc := resourceBranchRestriction().Schema["users"].Set

	// A UUID typed in mixed case and the same UUID in all-lowercase must hash
	// to the same bucket so that Terraform doesn't see them as different set
	// elements (which would produce a perpetual diff).
	mixedCase := setFunc("{C0FFEE00-C0FF-EEC0-FFEE-C0FFEEC0FFEE}")
	lowerCase := setFunc("{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}")
	if mixedCase != lowerCase {
		t.Errorf("users set func: mixed-case UUID hashed to %d, lowercase to %d; want equal", mixedCase, lowerCase)
	}

	// A plain username (not a UUID) must still produce a stable hash and must
	// differ from an unrelated value.
	h1 := setFunc("my-username")
	h2 := setFunc("other-username")
	if h1 == h2 {
		t.Errorf("users set func: distinct usernames hashed to the same value %d", h1)
	}
}

func TestBranchRestrictionGroupsSetFunc(t *testing.T) {
	setFunc := resourceBranchRestriction().Schema["groups"].Set

	// A UUID typed in mixed case and the same UUID in all-lowercase must hash
	// to the same bucket so that Terraform doesn't see them as different set
	// elements (which would produce a perpetual diff).
	mixedCase := setFunc(map[string]interface{}{"owner": "{C0FFEE00-C0FF-EEC0-FFEE-C0FFEEC0FFEE}", "slug": "my-group"})
	lowerCase := setFunc(map[string]interface{}{"owner": "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", "slug": "my-group"})
	if mixedCase != lowerCase {
		t.Errorf("groups set func: mixed-case UUID hashed to %d, lowercase to %d; want equal", mixedCase, lowerCase)
	}

	// Different slugs with the same owner must hash differently.
	h1 := setFunc(map[string]interface{}{"owner": "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", "slug": "group-a"})
	h2 := setFunc(map[string]interface{}{"owner": "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", "slug": "group-b"})
	if h1 == h2 {
		t.Errorf("groups set func: distinct slugs hashed to the same value %d", h1)
	}

	// Different owners with the same slug must also hash differently.
	h3 := setFunc(map[string]interface{}{"owner": "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", "slug": "my-group"})
	h4 := setFunc(map[string]interface{}{"owner": "{deadbeef-dead-beef-dead-beefdeadbeef}", "slug": "my-group"})
	if h3 == h4 {
		t.Errorf("groups set func: distinct owners hashed to the same value %d", h3)
	}

	// Pairs that would collide with a naive ";" separator must hash differently.
	// owner="a;b" slug="c" vs owner="a" slug="b;c" — distinct pairs, distinct hashes.
	hX := setFunc(map[string]interface{}{"owner": "a;b", "slug": "c"})
	hY := setFunc(map[string]interface{}{"owner": "a", "slug": "b;c"})
	if hX == hY {
		t.Errorf("groups set func: separator collision — {owner:\"a;b\",slug:\"c\"} and {owner:\"a\",slug:\"b;c\"} hashed to same value %d", hX)
	}
}

func TestFlattenBranchRestrictionUsers(t *testing.T) {
	cases := []struct {
		name  string
		input []bitbucket.Account
		want  []string
	}{
		{
			"nil slice",
			nil,
			[]string{},
		},
		{
			"empty slice",
			[]bitbucket.Account{},
			[]string{},
		},
		{
			"single user",
			[]bitbucket.Account{{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"}},
			[]string{"{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
		},
		{
			"multiple users",
			[]bitbucket.Account{
				{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
				{Uuid: "{deadbeef-dead-beef-dead-beefdeadbeef}"},
			},
			[]string{"{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", "{deadbeef-dead-beef-dead-beefdeadbeef}"},
		},
		{
			"falls back to username when uuid empty",
			[]bitbucket.Account{
				{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
				{Username: "legacy-username-with-no-uuid"},
			},
			[]string{"{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", "legacy-username-with-no-uuid"},
		},
		{
			"fully empty account omitted",
			[]bitbucket.Account{
				{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
				{},
			},
			[]string{"{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := flattenBranchRestrictionUsers(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("flattenBranchRestrictionUsers() = %v, want %v", got, tc.want)
			}
		})
	}
}

// makeGroupsSet builds a *schema.Set matching the groups TypeSet schema,
// seeded with the provided (owner, slug) pairs, for use in prevState arguments.
func makeGroupsSet(pairs []map[string]interface{}) *schema.Set {
	setFunc := resourceBranchRestriction().Schema["groups"].Set
	s := schema.NewSet(setFunc, nil)
	for _, m := range pairs {
		s.Add(m)
	}
	return s
}

func TestFlattenBranchRestrictionGroups(t *testing.T) {
	wsUUID := "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"
	ownerUUID := "{deadbeef-dead-beef-dead-beefdeadbeef}"

	cases := []struct {
		name      string
		input     []bitbucket.Group
		prevState *schema.Set
		want      []interface{}
	}{
		{
			"nil slice",
			nil,
			nil,
			[]interface{}{},
		},
		{
			"empty slice",
			[]bitbucket.Group{},
			nil,
			[]interface{}{},
		},
		{
			// The API may return group.Workspace with a UUID — preferred above all
			// other identifiers so users who configure owner as UUID see a stable
			// round-trip when this field is populated.
			"workspace uuid preferred over owner uuid",
			[]bitbucket.Group{
				{
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "ws-slug"},
					Owner:     &bitbucket.Account{Uuid: ownerUUID},
					Slug:      "my-group",
				},
			},
			nil,
			[]interface{}{
				map[string]interface{}{"owner": wsUUID, "slug": "my-group"},
			},
		},
		{
			// Regression: Bitbucket API returns group.Owner.Username = workspace slug,
			// group.Owner.Uuid = "", group.Workspace = nil (API omits Workspace field).
			// If prevState had a UUID owner for the same slug, preserve it.
			"prevState uuid preserved when API returns slug (regression)",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{Username: "sycle-corp"}, Slug: "administrators"},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": "{c73001e4-85bf-4fa1-a18e-f02a5fc392c4}", "slug": "administrators"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": "{c73001e4-85bf-4fa1-a18e-f02a5fc392c4}", "slug": "administrators"},
			},
		},
		{
			// Multiple groups: UUID owners in prevState are all preserved.
			"prevState uuid preserved for multiple groups",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{Username: "sycle-corp"}, Slug: "admins"},
				{Owner: &bitbucket.Account{Username: "sycle-corp"}, Slug: "developers"},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": wsUUID, "slug": "admins"},
				{"owner": wsUUID, "slug": "developers"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": wsUUID, "slug": "admins"},
				map[string]interface{}{"owner": wsUUID, "slug": "developers"},
			},
		},
		{
			// If prevState has a slug owner and API returns slug, slug is used.
			// No UUID in prevState → no substitution.
			"slug in prevState stays slug",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{Username: "sycle-corp"}, Slug: "my-group"},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": "sycle-corp", "slug": "my-group"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": "sycle-corp", "slug": "my-group"},
			},
		},
		{
			// New group (not in prevState): slug is used as-is.
			"new group not in prevState uses api value",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{Username: "sycle-corp"}, Slug: "new-group"},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": wsUUID, "slug": "other-group"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": "sycle-corp", "slug": "new-group"},
			},
		},
		{
			// When workspace UUID is absent, fall back to owner UUID.
			"owner uuid used when workspace uuid empty",
			[]bitbucket.Group{
				{
					Workspace: &bitbucket.Workspace{Slug: "ws-slug"},
					Owner:     &bitbucket.Account{Uuid: ownerUUID},
					Slug:      "my-group",
				},
			},
			nil,
			[]interface{}{
				map[string]interface{}{"owner": ownerUUID, "slug": "my-group"},
			},
		},
		{
			// When both UUIDs are absent and no prevState, prefer workspace slug.
			"workspace slug preferred over owner username (no prevState)",
			[]bitbucket.Group{
				{
					Workspace: &bitbucket.Workspace{Slug: "ws-slug"},
					Owner:     &bitbucket.Account{Username: "owner-username"},
					Slug:      "my-group",
				},
			},
			nil,
			[]interface{}{
				map[string]interface{}{"owner": "ws-slug", "slug": "my-group"},
			},
		},
		{
			// Last-resort fallback: no workspace, owner has only username.
			"owner username fallback when workspace nil",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{Username: "legacy-owner"}, Slug: "my-group"},
			},
			nil,
			[]interface{}{
				map[string]interface{}{"owner": "legacy-owner", "slug": "my-group"},
			},
		},
		{
			// A group with no resolvable owner cannot be expressed in HCL
			// (owner is Required), so it is dropped rather than written with
			// owner:"" which would guarantee a persistent diff.
			"nil owner and nil workspace omitted",
			[]bitbucket.Group{
				{Owner: nil, Workspace: nil, Slug: "my-group"},
			},
			nil,
			[]interface{}{},
		},
		{
			// Same: all identifier fields empty → must be dropped.
			"fully empty owner and workspace omitted",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{}, Workspace: &bitbucket.Workspace{}, Slug: "my-group"},
			},
			nil,
			[]interface{}{},
		},
		{
			// CLAUDE.md requires flatten-helper tests to cover a mix of valid
			// (UUID) and legacy-format (username) values in the same collection.
			"mix: prevState UUID preserved + new slug group",
			[]bitbucket.Group{
				{Owner: &bitbucket.Account{Username: "sycle-corp"}, Slug: "uuid-group"},
				{Owner: &bitbucket.Account{Username: "legacy-owner"}, Slug: "legacy-group"},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": wsUUID, "slug": "uuid-group"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": wsUUID, "slug": "uuid-group"},
				map[string]interface{}{"owner": "legacy-owner", "slug": "legacy-group"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := flattenBranchRestrictionGroups(tc.input, tc.prevState)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("flattenBranchRestrictionGroups() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAccountIdentifier(t *testing.T) {
	cases := []struct {
		name  string
		input bitbucket.Account
		want  string
	}{
		{"uuid present", bitbucket.Account{Uuid: "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}", Username: "legacy"}, "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"},
		{"username only", bitbucket.Account{Username: "legacy-username"}, "legacy-username"},
		{"both empty", bitbucket.Account{}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := accountIdentifier(tc.input)
			if got != tc.want {
				t.Errorf("accountIdentifier(%+v) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestGroupOwnerIdentifier(t *testing.T) {
	wsUUID := "{c0ffee00-c0ff-eec0-ffee-c0ffeec0ffee}"
	ownerUUID := "{deadbeef-dead-beef-dead-beefdeadbeef}"

	cases := []struct {
		name  string
		input bitbucket.Group
		want  string
	}{
		{
			"workspace uuid preferred",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "ws-slug"},
				Owner:     &bitbucket.Account{Uuid: ownerUUID, Username: "owner-slug"},
			},
			wsUUID,
		},
		{
			// Regression: API returns Owner.Username = workspace slug, Owner.Uuid = "".
			// If group.Workspace.Uuid is present it must win.
			"workspace uuid beats owner username (regression)",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "sycle-corp"},
				Owner:     &bitbucket.Account{Username: "sycle-corp"},
			},
			wsUUID,
		},
		{
			"owner uuid when workspace uuid absent",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{Slug: "ws-slug"},
				Owner:     &bitbucket.Account{Uuid: ownerUUID},
			},
			ownerUUID,
		},
		{
			"workspace slug when both uuids absent",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{Slug: "ws-slug"},
				Owner:     &bitbucket.Account{Username: "owner-username"},
			},
			"ws-slug",
		},
		{
			"owner username last resort",
			bitbucket.Group{
				Owner: &bitbucket.Account{Username: "legacy-owner"},
			},
			"legacy-owner",
		},
		{
			"nil workspace and nil owner returns empty",
			bitbucket.Group{},
			"",
		},
		{
			"empty workspace and empty owner returns empty",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{},
				Owner:     &bitbucket.Account{},
			},
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := groupOwnerIdentifier(tc.input)
			if got != tc.want {
				t.Errorf("groupOwnerIdentifier() = %q, want %q", got, tc.want)
			}
		})
	}
}

func testAccBitbucketBranchRestrictionConfig(testUser, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "force"
  pattern    = "master"
}
`, testUser, rName)
}

func testAccBitbucketBranchRestrictionModelConfig(testUser, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branch_restriction" "test" {
  owner             = %[1]q
  repository        = bitbucket_repository.test.name
  kind              = "force"
  branch_match_kind = "branching_model"
  branch_type       = "production"
}
`, testUser, rName)
}

func testAccBitbucketBranchRestrictionUsersConfig(testUser, rName string) string {
	return fmt.Sprintf(`
data "bitbucket_current_user" "test" {}

resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "push"
  pattern    = "master"
  users      = [data.bitbucket_current_user.test.uuid]
}
`, testUser, rName)
}

func testAccBitbucketBranchRestrictionGroupsConfig(testUser, rName string) string {
	return fmt.Sprintf(`
data "bitbucket_workspace" "test" {
  workspace = %[1]q
}

resource "bitbucket_group" "test" {
  # The 1.0/groups API requires the workspace slug, not the UUID.
  # data.bitbucket_workspace.test.id returns the UUID, so .slug is used here.
  # The branch restriction groups.owner still uses .id (UUID) — the 2.0 API
  # returns and accepts UUIDs for that field.
  workspace = data.bitbucket_workspace.test.slug
  name      = %[2]q
}

resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "push"
  pattern    = "master"
  groups {
    owner = data.bitbucket_workspace.test.id
    slug  = bitbucket_group.test.slug
  }
}
`, testUser, rName)
}

// testAccBitbucketBranchRestrictionGroupsExistingConfig creates a repo, grants
// the pre-existing group write access (required before it can be used in a push
// restriction), then creates a push restriction referencing the group.
// groups.owner uses the workspace UUID (data.bitbucket_workspace.test.id) so
// the test verifies the UUID round-trip through the 2.0 API.
func testAccBitbucketBranchRestrictionGroupsExistingConfig(testUser, rName, groupSlug string) string {
	return fmt.Sprintf(`
data "bitbucket_workspace" "test" {
  workspace = %[1]q
}

resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_repository_group_permission" "test" {
  workspace  = %[1]q
  repo_slug  = bitbucket_repository.test.name
  group_slug = %[3]q
  permission = "write"
}

resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "push"
  pattern    = "master"

  groups {
    owner = data.bitbucket_workspace.test.id
    slug  = %[3]q
  }

  depends_on = [bitbucket_repository_group_permission.test]
}
`, testUser, rName, groupSlug)
}

func testAccBitbucketBranchRestrictionLegacyUsernameConfig(testUser, rName, legacyUsername string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "push"
  pattern    = "master"
  users      = [%[3]q]
}
`, testUser, rName, legacyUsername)
}

func testAccCheckBitbucketBranchRestrictionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(Clients).genClient
	brApi := client.ApiClient.BranchRestrictionsApi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "bitbucket_branch_restriction" {
			continue
		}

		_, res, err := brApi.RepositoriesWorkspaceRepoSlugBranchRestrictionsIdGet(client.AuthContext,
			url.PathEscape(rs.Primary.ID),
			rs.Primary.Attributes["repository"], rs.Primary.Attributes["owner"])

		if err == nil {
			return fmt.Errorf("The resource was found should have errored")
		}

		if res.StatusCode != http.StatusNotFound {
			return fmt.Errorf("BranchRestriction still exists")
		}
	}

	return nil
}

func testAccCheckBitbucketBranchRestrictionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("No BranchRestriction ID is set")
		}
		return nil
	}
}

func testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("Not found: %s", resourceName)
		}
		return fmt.Sprintf("%s/%s/%s", rs.Primary.Attributes["owner"], rs.Primary.Attributes["repository"], rs.Primary.ID), nil
	}
}
