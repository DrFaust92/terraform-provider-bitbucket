package bitbucket

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-cty/cty"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
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

func TestAccBitbucketBranchRestriction_requireAllCommentsResolved(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionKindConfig(testUser, rName, "require_all_comments_resolved"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "kind", "require_all_comments_resolved"),
					resource.TestCheckResourceAttr(resourceName, "pattern", "master"),
				),
			},
			{
				Config:   testAccBitbucketBranchRestrictionKindConfig(testUser, rName, "require_all_comments_resolved"),
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

func TestBranchRestrictionKindValidation(t *testing.T) {
	validate := resourceBranchRestriction().Schema["kind"].ValidateFunc

	if _, errs := validate("require_all_comments_resolved", "kind"); len(errs) > 0 {
		t.Errorf("expected require_all_comments_resolved to be valid, got %v", errs)
	}
	if _, errs := validate("not_a_kind", "kind"); len(errs) == 0 {
		t.Error("expected not_a_kind to be rejected")
	}
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
					// The 2.0 API returns both owner.uuid and workspace.uuid in the group
					// object; groups.owner round-trips as the workspace UUID here since
					// that's the form this config uses.
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
//   - groups.owner in state matches the workspace UUID when configured as a UUID
//   - groups.owner in state matches the workspace slug when configured as a slug
//   - a plan immediately after each apply is empty (no perpetual diff)
//   - ImportStateVerify passes
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
				// Apply with the slug owner form first — this is also the
				// default flattenBranchRestrictionGroups falls back to when
				// there's no prior state to preserve (e.g. on import, below),
				// so this ordering keeps the import step's comparison baseline
				// consistent with that default.
				Config: testAccBitbucketBranchRestrictionGroupsExistingSlugOwnerConfig(testUser, rName, groupSlug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "groups.0.slug", groupSlug),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "groups.*.owner", "data.bitbucket_workspace.test", "slug"),
				),
			},
			{
				// Slug round-trip: a plan immediately after apply must be empty.
				Config:   testAccBitbucketBranchRestrictionGroupsExistingSlugOwnerConfig(testUser, rName, groupSlug),
				PlanOnly: true,
			},
			{
				// Import has no prior state at all to preserve a configured form
				// from, so it reads back groupOwnerIdentifier's bare default (the
				// slug) — which is why this step runs right after a slug-owner
				// apply above, rather than after a UUID-owner one.
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				// Switch the config to use the workspace UUID instead of the
				// slug. This is a real config change, so a one-time diff to
				// converge on the UUID is expected and fine.
				Config: testAccBitbucketBranchRestrictionGroupsExistingConfig(testUser, rName, groupSlug),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckTypeSetElemAttrPair(resourceName, "groups.*.owner", "data.bitbucket_workspace.test", "id"),
				),
			},
			{
				// UUID round-trip: once in state, the UUID form must not flip
				// back to the slug on a subsequent Read.
				Config:   testAccBitbucketBranchRestrictionGroupsExistingConfig(testUser, rName, groupSlug),
				PlanOnly: true,
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

// TestAccBitbucketBranchRestriction_reviewGroupApprovals covers the
// require_review_group_approvals_to_merge kind, an on/off restriction.
// Bitbucket rejects it with HTTP 400 when value, users or groups are set, and
// the provider catches that at plan time.
func TestAccBitbucketBranchRestriction_reviewGroupApprovals(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders, //nolint:staticcheck // pre-existing repo-wide pattern; ProviderFactories migration is out of scope
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccBitbucketBranchRestrictionReviewGroupApprovalsInvalidConfig(testUser, rName, "value = 1"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`does not accept value`),
			},
			{
				Config:      testAccBitbucketBranchRestrictionReviewGroupApprovalsInvalidConfig(testUser, rName, fmt.Sprintf("groups {\n    owner = %q\n    slug  = \"any-group\"\n  }", testUser)),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`does not accept users or groups`),
			},
			{
				Config: testAccBitbucketBranchRestrictionReviewGroupApprovalsConfig(testUser, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "kind", "require_review_group_approvals_to_merge"),
					resource.TestCheckResourceAttr(resourceName, "value", "0"),
					resource.TestCheckResourceAttr(resourceName, "users.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "groups.#", "0"),
				),
			},
			{
				Config:   testAccBitbucketBranchRestrictionReviewGroupApprovalsConfig(testUser, rName),
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

// TestAccBitbucketBranchRestriction_pipelinesSourceBranches verifies that
// pipelines_source_branches round-trips on create, import and update,
// including being cleared by removing it from config.
func TestAccBitbucketBranchRestriction_pipelinesSourceBranches(t *testing.T) {
	rName := acctest.RandomWithPrefix("tf-test")
	testUser := os.Getenv("BITBUCKET_TEAM")
	resourceName := "bitbucket_branch_restriction.test"

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders, //nolint:staticcheck // pre-existing repo-wide pattern; ProviderFactories migration is out of scope
		CheckDestroy: testAccCheckBitbucketBranchRestrictionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccBitbucketBranchRestrictionPipelinesSourceBranchesConfig(testUser, rName, `["develop", "staging"]`),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBitbucketBranchRestrictionExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "pipelines_source_branches.#", "2"),
					resource.TestCheckTypeSetElemAttr(resourceName, "pipelines_source_branches.*", "develop"),
					resource.TestCheckTypeSetElemAttr(resourceName, "pipelines_source_branches.*", "staging"),
				),
			},
			{
				Config:   testAccBitbucketBranchRestrictionPipelinesSourceBranchesConfig(testUser, rName, `["develop", "staging"]`),
				PlanOnly: true,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateIdFunc: testAccCheckBitbucketBranchRestrictionImportStateIdFunc(resourceName),
				ImportStateVerify: true,
			},
			{
				Config: testAccBitbucketBranchRestrictionPipelinesSourceBranchesConfig(testUser, rName, `["develop"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pipelines_source_branches.#", "1"),
					resource.TestCheckTypeSetElemAttr(resourceName, "pipelines_source_branches.*", "develop"),
				),
			},
			{
				// Removing the attribute must clear it on the API side too,
				// not leave the old list behind (which would diff forever).
				Config: testAccBitbucketBranchRestrictionPipelinesSourceBranchesConfig(testUser, rName, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "pipelines_source_branches.#", "0"),
				),
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

func TestCreateBranchRestrictionPipelinesSourceBranches(t *testing.T) {
	raw := map[string]interface{}{
		"owner":                     "myteam",
		"repository":                "my-repo",
		"kind":                      "push",
		"pattern":                   "main",
		"pipelines_source_branches": []interface{}{"develop", "staging"},
	}

	d := schema.TestResourceDataRaw(t, resourceBranchRestriction().Schema, raw)
	got := createBranchRestriction(d)

	if got.PipelinesSourceBranches == nil {
		t.Fatal("PipelinesSourceBranches = nil, want the configured branches")
	}
	want := map[string]bool{"develop": true, "staging": true}
	if len(*got.PipelinesSourceBranches) != len(want) {
		t.Fatalf("PipelinesSourceBranches = %v, want (any order) [develop staging]", *got.PipelinesSourceBranches)
	}
	for _, b := range *got.PipelinesSourceBranches {
		if !want[b] {
			t.Errorf("unexpected source branch %q", b)
		}
	}
}

func TestCreateBranchRestrictionOmitsUnsetPipelinesSourceBranches(t *testing.T) {
	raw := map[string]interface{}{
		"owner":      "myteam",
		"repository": "my-repo",
		"kind":       "push",
		"pattern":    "main",
	}

	d := schema.TestResourceDataRaw(t, resourceBranchRestriction().Schema, raw)
	got := createBranchRestriction(d)

	if got.PipelinesSourceBranches != nil {
		t.Errorf("PipelinesSourceBranches = %v, want nil when unset", *got.PipelinesSourceBranches)
	}
}

// TestBranchRestrictionBodyJSON pins the wire format: the embedded generated
// model's fields stay at the top level, pipelines_source_branches is omitted
// when nil and sent as [] when pointing at an empty slice (to clear it).
func TestBranchRestrictionBodyJSON(t *testing.T) {
	empty := []string{}
	branches := []string{"develop"}

	cases := []struct {
		name      string
		branches  *[]string
		wantField bool
		wantValue []string
	}{
		{name: "nil omits field", branches: nil, wantField: false},
		{name: "empty sends []", branches: &empty, wantField: true, wantValue: []string{}},
		{name: "values sent", branches: &branches, wantField: true, wantValue: []string{"develop"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := branchRestrictionBody{
				Branchrestriction: bitbucket.Branchrestriction{
					Kind:            "require_review_group_approvals_to_merge",
					BranchMatchKind: "glob",
					Pattern:         "main",
					Value:           2,
				},
				PipelinesSourceBranches: tc.branches,
			}

			data, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}

			var m map[string]json.RawMessage
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			if string(m["kind"]) != `"require_review_group_approvals_to_merge"` {
				t.Errorf("kind = %s, want top-level kind", m["kind"])
			}
			if string(m["value"]) != "2" {
				t.Errorf("value = %s, want 2", m["value"])
			}

			raw, ok := m["pipelines_source_branches"]
			if ok != tc.wantField {
				t.Fatalf("pipelines_source_branches present = %v, want %v (json: %s)", ok, tc.wantField, data)
			}
			if !tc.wantField {
				return
			}
			var got []string
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("Unmarshal pipelines_source_branches: %v", err)
			}
			if got == nil || !reflect.DeepEqual(got, tc.wantValue) {
				t.Errorf("pipelines_source_branches = %s, want %v", raw, tc.wantValue)
			}
		})
	}
}

// TestBranchRestrictionBodyDecode checks that a response decodes into both
// the embedded model's fields and the added pipelines_source_branches field.
func TestBranchRestrictionBodyDecode(t *testing.T) {
	const resp = `{"id": 42, "kind": "push", "branch_match_kind": "glob", "pattern": "main",
		"pipelines_source_branches": ["develop"], "users": [], "groups": [], "type": "branchrestriction"}`

	var got branchRestrictionBody
	if err := json.Unmarshal([]byte(resp), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Id != 42 || got.Kind != "push" || got.Pattern != "main" {
		t.Errorf("embedded fields = {Id:%d Kind:%q Pattern:%q}, want {42 push main}", got.Id, got.Kind, got.Pattern)
	}
	if got.PipelinesSourceBranches == nil || !reflect.DeepEqual(*got.PipelinesSourceBranches, []string{"develop"}) {
		t.Errorf("PipelinesSourceBranches = %v, want [develop]", got.PipelinesSourceBranches)
	}
}

func TestBranchRestrictionsEndpoint(t *testing.T) {
	if got, want := branchRestrictionsEndpoint("myteam", "my-repo", ""), "2.0/repositories/myteam/my-repo/branch-restrictions"; got != want {
		t.Errorf("collection endpoint = %q, want %q", got, want)
	}
	if got, want := branchRestrictionsEndpoint("myteam", "my-repo", "42"), "2.0/repositories/myteam/my-repo/branch-restrictions/42"; got != want {
		t.Errorf("item endpoint = %q, want %q", got, want)
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

func TestValidateBranchRestrictionFields(t *testing.T) {
	const reviewGroup = "require_review_group_approvals_to_merge"
	cases := []struct {
		kind          string
		valueSet      bool
		users, groups int
		wantErr       string
	}{
		{reviewGroup, false, 0, 0, ""},
		{reviewGroup, true, 0, 0, "does not accept value"},
		{reviewGroup, false, 0, 1, "does not accept users or groups"},
		{reviewGroup, false, 1, 0, "does not accept users or groups"},
		{"push", true, 1, 1, ""},
		{"require_approvals_to_merge", true, 0, 0, ""},
	}

	for _, tc := range cases {
		err := validateBranchRestrictionFields(tc.kind, tc.valueSet, tc.users, tc.groups)
		if tc.wantErr == "" && err != nil || tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
			t.Errorf("validateBranchRestrictionFields(%q, %t, %d, %d) = %v, want error containing %q", tc.kind, tc.valueSet, tc.users, tc.groups, err, tc.wantErr)
		}
	}

	if resourceBranchRestriction().CustomizeDiff == nil {
		t.Error("bitbucket_branch_restriction must run the field check at plan time")
	}
}

func TestConfigAttrSet(t *testing.T) {
	obj := func(v cty.Value) cty.Value { return cty.ObjectVal(map[string]cty.Value{"value": v}) }
	cases := map[string]struct {
		config cty.Value
		want   bool
	}{
		"zero":     {obj(cty.NumberIntVal(0)), true},
		"non-zero": {obj(cty.NumberIntVal(2)), true},
		"null":     {obj(cty.NullVal(cty.Number)), false},
		"no attr":  {cty.EmptyObjectVal, false},
		"unknown":  {cty.UnknownVal(cty.EmptyObject), false},
	}
	for name, tc := range cases {
		if got := configAttrSet(tc.config, "value"); got != tc.want {
			t.Errorf("%s: configAttrSet = %t, want %t", name, got, tc.want)
		}
	}
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
			// New group (not in prevState): the API returns both a slug and a
			// UUID, and the slug is used since it's the default/preferred form.
			"new group with both forms defaults to slug",
			[]bitbucket.Group{
				{
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "ws-slug"},
					Owner:     &bitbucket.Account{Uuid: ownerUUID},
					Slug:      "my-group",
				},
			},
			nil,
			[]interface{}{
				map[string]interface{}{"owner": "ws-slug", "slug": "my-group"},
			},
		},
		{
			// New group with no slug available at all: falls back to the UUID.
			"new group with uuid only falls back to uuid",
			[]bitbucket.Group{
				{
					Workspace: &bitbucket.Workspace{Uuid: wsUUID},
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
			// KEY REGRESSION CASE: prevState has the slug, and the API returns
			// both a UUID and the same slug for the group owner. The slug must
			// be preserved — not flipped to the UUID — since the slug is what
			// the user's config and state already use.
			"prevState slug preserved when API also returns a uuid",
			[]bitbucket.Group{
				{
					Owner:     &bitbucket.Account{Uuid: wsUUID, Username: "myteam"},
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "myteam"},
					Slug:      "administrators",
				},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": "myteam", "slug": "administrators"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": "myteam", "slug": "administrators"},
			},
		},
		{
			// Opposite direction: prevState has the UUID (e.g. from before this
			// fix, or a UUID-based config), API returns both forms — the UUID
			// must be preserved, not flipped to the slug.
			"prevState uuid preserved when API also returns a slug",
			[]bitbucket.Group{
				{
					Owner:     &bitbucket.Account{Uuid: wsUUID, Username: "myteam"},
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "myteam"},
					Slug:      "administrators",
				},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": wsUUID, "slug": "administrators"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": wsUUID, "slug": "administrators"},
			},
		},
		{
			// prevState owner matches neither the slug nor the UUID the API
			// returned for this group slug (e.g. stale/unrelated value) — falls
			// back to the default (slug) rather than keeping a mismatched value.
			"prevState owner matching neither form falls back to slug default",
			[]bitbucket.Group{
				{
					Owner:     &bitbucket.Account{Uuid: wsUUID, Username: "myteam"},
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "myteam"},
					Slug:      "administrators",
				},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": "some-other-value", "slug": "administrators"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": "myteam", "slug": "administrators"},
			},
		},
		{
			// Case-insensitive match: prevState's exact casing is preserved
			// even though it differs from the API's canonical lowercase UUID.
			"prevState uuid casing preserved on case-insensitive match",
			[]bitbucket.Group{
				{
					Owner:     &bitbucket.Account{Uuid: wsUUID, Username: "myteam"},
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "myteam"},
					Slug:      "administrators",
				},
			},
			makeGroupsSet([]map[string]interface{}{
				{"owner": strings.ToUpper(wsUUID), "slug": "administrators"},
			}),
			[]interface{}{
				map[string]interface{}{"owner": strings.ToUpper(wsUUID), "slug": "administrators"},
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
			// (preserved-from-state) and legacy-format (no prevState, username
			// only) values in the same collection.
			"mix: prevState match preserved + brand new username-only group",
			[]bitbucket.Group{
				{
					Owner:     &bitbucket.Account{Uuid: wsUUID, Username: "myteam"},
					Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "myteam"},
					Slug:      "uuid-group",
				},
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
			// The API returns both forms for every group; the slug is the
			// default even when a UUID is also present, since that's the
			// form the docs call preferred and most existing configs use.
			"workspace slug preferred even when uuids are also present",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{Uuid: wsUUID, Slug: "ws-slug"},
				Owner:     &bitbucket.Account{Uuid: ownerUUID, Username: "owner-slug"},
			},
			"ws-slug",
		},
		{
			"owner username used as slug when workspace is nil",
			bitbucket.Group{
				Owner: &bitbucket.Account{Uuid: ownerUUID, Username: "owner-slug"},
			},
			"owner-slug",
		},
		{
			"workspace uuid used when no slug is available",
			bitbucket.Group{
				Workspace: &bitbucket.Workspace{Uuid: wsUUID},
				Owner:     &bitbucket.Account{Uuid: ownerUUID},
			},
			wsUUID,
		},
		{
			"owner uuid used when workspace is absent entirely",
			bitbucket.Group{
				Owner: &bitbucket.Account{Uuid: ownerUUID},
			},
			ownerUUID,
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

func testAccBitbucketBranchRestrictionKindConfig(testUser, rName, kind string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}
resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = %[3]q
  pattern    = "master"
}
`, testUser, rName, kind)
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

// testAccBitbucketBranchRestrictionGroupsExistingSlugOwnerConfig is identical
// to testAccBitbucketBranchRestrictionGroupsExistingConfig except groups.owner
// uses the workspace slug (data.bitbucket_workspace.test.slug) instead of the
// UUID, to verify the slug form round-trips stably too.
func testAccBitbucketBranchRestrictionGroupsExistingSlugOwnerConfig(testUser, rName, groupSlug string) string {
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
    owner = data.bitbucket_workspace.test.slug
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

func testAccBitbucketBranchRestrictionReviewGroupApprovalsConfig(testUser, rName string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "require_review_group_approvals_to_merge"
  pattern    = "master"
}
`, testUser, rName)
}

// testAccBitbucketBranchRestrictionReviewGroupApprovalsInvalidConfig adds
// extra (a field the kind doesn't accept) to a
// require_review_group_approvals_to_merge restriction, which the provider must
// reject at plan time. Nothing is looked up, so placeholder values will do.
func testAccBitbucketBranchRestrictionReviewGroupApprovalsInvalidConfig(testUser, rName, extra string) string {
	return fmt.Sprintf(`
resource "bitbucket_repository" "test" {
  owner = %[1]q
  name  = %[2]q
}

resource "bitbucket_branch_restriction" "test" {
  owner      = %[1]q
  repository = bitbucket_repository.test.name
  kind       = "require_review_group_approvals_to_merge"
  pattern    = "master"

  %[3]s
}
`, testUser, rName, extra)
}

// testAccBitbucketBranchRestrictionPipelinesSourceBranchesConfig creates a
// push restriction with pipelines_source_branches set to sourceBranches (an
// HCL list literal), or without the attribute when sourceBranches is "", so a
// later step can drop it and update the same restriction in place. It uses
// push because Bitbucket rejects the field on kinds such as force ("kind
// force requires pipelines_source_branches to be empty, null, or omitted").
func testAccBitbucketBranchRestrictionPipelinesSourceBranchesConfig(testUser, rName, sourceBranches string) string {
	sourceBranchesLine := ""
	if sourceBranches != "" {
		sourceBranchesLine = "pipelines_source_branches = " + sourceBranches
	}

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
  %[3]s
}
`, testUser, rName, sourceBranchesLine)
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
