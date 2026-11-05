package bitbucket

import (
	"context"
	"fmt"
	"log"

	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/DrFaust92/bitbucket-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// bitbucketUUIDPattern matches Bitbucket's canonical curly-brace-wrapped UUID format.
var bitbucketUUIDPattern = regexp.MustCompile(`^\{[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\}$`)

// BranchRestriction is the data we need to send to create a new branch restriction for the repository
type BranchRestriction struct {
	ID              int     `json:"id,omitempty"`
	Kind            string  `json:"kind,omitempty"`
	BranchMatchkind string  `json:"branch_match_kind,omitempty"`
	BranchType      string  `json:"branch_type,omitempty"`
	Pattern         string  `json:"pattern,omitempty"`
	Value           int     `json:"value,omitempty"`
	Users           []User  `json:"users,omitempty"`
	Groups          []Group `json:"groups,omitempty"`
}

// User is just the user struct we want to use for BranchRestrictions
type User struct {
	Username string `json:"username,omitempty"`
}

// Group is the group we want to add to a branch restriction
type Group struct {
	Slug  string `json:"slug,omitempty"`
	Owner User   `json:"owner,omitempty"`
}

func resourceBranchRestriction() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceBranchRestrictionsCreate,
		ReadContext:   resourceBranchRestrictionsRead,
		UpdateContext: resourceBranchRestrictionsUpdate,
		DeleteContext: resourceBranchRestrictionsDelete,
		Importer: &schema.ResourceImporter{
			State: func(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				idParts := strings.Split(d.Id(), "/")
				if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
					return nil, fmt.Errorf("unexpected format of ID (%q), expected OWNER/REPO/BRANCH-RESTRICTION-ID", d.Id())
				}
				d.SetId(idParts[2])
				d.Set("owner", idParts[0])
				d.Set("repository", idParts[1])
				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			"owner": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"repository": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"kind": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"allow_auto_merge_when_builds_pass",
					"delete",
					"enforce_merge_checks",
					"force",
					"push",
					"require_all_dependencies_merged",
					"require_approvals_to_merge",
					"require_commits_behind",
					"require_default_reviewer_approvals_to_merge",
					"require_no_changes_requested",
					"require_passing_builds_to_merge",
					"require_tasks_to_be_completed",
					"reset_pullrequest_approvals_on_change",
					"reset_pullrequest_changes_requested_on_change",
					"restrict_merges",
					"smart_reset_pullrequest_approvals",
				}, false),
			},
			"branch_match_kind": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "glob",
				ValidateFunc: validation.StringInSlice([]string{"branching_model", "glob"}, false),
			},
			"pattern": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"branch_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"feature", "bugfix", "release", "hotfix", "development", "production"}, false),
			},
			"users": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				// Normalize to lowercase before hashing so that a UUID typed in
				// mixed case (e.g. {C0FFEE00-...}) lands in the same set bucket
				// as the lowercase form the API returns, preventing a perpetual
				// diff for users who supply upper- or mixed-case UUIDs.
				Set: func(v interface{}) int {
					return schema.HashString(strings.ToLower(v.(string)))
				},
			},
			"groups": {
				Type: schema.TypeSet,
				// Normalize owner to lowercase before hashing so that a UUID typed
				// in mixed case (e.g. {C0FFEE00-...}) lands in the same set bucket
				// as the lowercase form the API returns, preventing a perpetual diff
				// for users who supply upper- or mixed-case UUIDs.  DiffSuppressFunc
				// on the owner attribute alone has no effect on TypeSet hash
				// computation, so this custom Set func is required.
				Set: func(v interface{}) int {
					m := v.(map[string]interface{})
					return schema.HashString(strings.ToLower(m["owner"].(string)) + "\x00" + m["slug"].(string))
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"owner": {
							Type:     schema.TypeString,
							Required: true,
							// Belt-and-suspenders: also suppress attribute-level diffs
							// caused by UUID casing so plan output stays clean even if
							// the hash normalization above misses an edge case.
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return strings.EqualFold(old, new)
							},
						},
						"slug": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Optional: true,
			},

			"value": {
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func createBranchRestriction(d *schema.ResourceData) *bitbucket.Branchrestriction {

	users := make([]bitbucket.Account, 0, d.Get("users").(*schema.Set).Len())

	for _, item := range d.Get("users").(*schema.Set).List() {
		users = append(users, expandBranchRestrictionUser(item.(string)))
	}

	groups := make([]bitbucket.Group, 0, d.Get("groups").(*schema.Set).Len())

	for _, item := range d.Get("groups").(*schema.Set).List() {
		m := item.(map[string]interface{})

		owner := expandBranchRestrictionUser(m["owner"].(string))

		group := bitbucket.Group{
			Owner: &owner,
			Slug:  m["slug"].(string),
		}

		groups = append(groups, group)
	}

	restict := &bitbucket.Branchrestriction{
		Kind:   d.Get("kind").(string),
		Value:  int32(d.Get("value").(int)),
		Users:  users,
		Groups: groups,
	}

	if v, ok := d.GetOk("pattern"); ok {
		restict.Pattern = v.(string)
	}

	if v, ok := d.GetOk("branch_type"); ok {
		restict.BranchType = v.(string)
	}

	if v, ok := d.GetOk("branch_match_kind"); ok {
		restict.BranchMatchKind = v.(string)
	}

	return restict
}

func resourceBranchRestrictionsCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	brApi := c.ApiClient.BranchRestrictionsApi
	branchRestriction := createBranchRestriction(d)

	repo := d.Get("repository").(string)
	workspace := d.Get("owner").(string)
	branchRestrictionReq, res, err := brApi.RepositoriesWorkspaceRepoSlugBranchRestrictionsPost(c.AuthContext, *branchRestriction, repo, workspace)
	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%v", branchRestrictionReq.Id))

	return resourceBranchRestrictionsRead(ctx, d, m)
}

func resourceBranchRestrictionsRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	brApi := c.ApiClient.BranchRestrictionsApi

	brRes, res, err := brApi.RepositoriesWorkspaceRepoSlugBranchRestrictionsIdGet(c.AuthContext, url.PathEscape(d.Id()),
		d.Get("repository").(string), d.Get("owner").(string))

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Branch Restrictions (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	// Warn when an existing resource (not a brand-new Create) still has a
	// legacy username-format user in state. Bitbucket's GDPR changes removed
	// usernames from account responses, so the API accepts them on write but
	// never echoes them back on read — this resource will keep diffing until
	// the config is updated to use the UUIDs now in state.
	// See: github.com/DrFaust92/terraform-provider-bitbucket/issues/173
	if !d.IsNewResource() && len(brRes.Users) > 0 {
		hasLegacyUsername := false
		for _, raw := range d.Get("users").(*schema.Set).List() {
			if !bitbucketUUIDPattern.MatchString(raw.(string)) {
				hasLegacyUsername = true
				break
			}
		}
		if hasLegacyUsername {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "branch_restriction: legacy username(s) in users will keep diffing",
				Detail: "The Bitbucket API no longer returns usernames, so this resource's " +
					"users will show a diff on every plan/apply until the config is updated " +
					"to use the UUIDs now in state. Add `lifecycle { ignore_changes = [users] }` " +
					"to suppress it if you can't migrate right away.",
			})
		}
	}

	d.SetId(fmt.Sprintf("%v", brRes.Id))
	d.Set("kind", brRes.Kind)
	d.Set("pattern", brRes.Pattern)
	d.Set("value", brRes.Value)
	if err := d.Set("users", flattenBranchRestrictionUsers(brRes.Users)); err != nil {
		return diag.FromErr(fmt.Errorf("error setting users: %w", err))
	}
	if err := d.Set("groups", flattenBranchRestrictionGroups(brRes.Groups, d.Get("groups").(*schema.Set))); err != nil {
		return diag.FromErr(fmt.Errorf("error setting groups: %w", err))
	}
	d.Set("branch_type", brRes.BranchType)
	d.Set("branch_match_kind", brRes.BranchMatchKind)

	return diags
}

func resourceBranchRestrictionsUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	brApi := c.ApiClient.BranchRestrictionsApi
	branchRestriction := createBranchRestriction(d)

	_, res, err := brApi.RepositoriesWorkspaceRepoSlugBranchRestrictionsIdPut(c.AuthContext,
		*branchRestriction, url.PathEscape(d.Id()),
		d.Get("repository").(string), d.Get("owner").(string))

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return resourceBranchRestrictionsRead(ctx, d, m)
}

func resourceBranchRestrictionsDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(Clients).genClient
	brApi := c.ApiClient.BranchRestrictionsApi

	res, err := brApi.RepositoriesWorkspaceRepoSlugBranchRestrictionsIdDelete(c.AuthContext, url.PathEscape(d.Id()),
		d.Get("repository").(string), d.Get("owner").(string))

	if res != nil && res.StatusCode == http.StatusNotFound {
		log.Printf("[WARN] Branch Restrictions (%s) not found, removing from state", d.Id())
		return nil
	}

	if err := handleClientError(res, err); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// expandBranchRestrictionUser builds the Account to send for a users or
// groups-owner entry. Bitbucket has deprecated legacy usernames, so a
// UUID-shaped entry is sent as Uuid (normalized to lowercase to match the
// canonical form the API returns, so Read converges with it); anything else
// is sent as Username, preserving existing configs.
func expandBranchRestrictionUser(item string) bitbucket.Account {
	if bitbucketUUIDPattern.MatchString(item) {
		return bitbucket.Account{Uuid: strings.ToLower(item)}
	}

	return bitbucket.Account{Username: item}
}

// accountIdentifier returns the identifier to report in state for an
// account: its uuid if present, else its (deprecated) username, else "".
func accountIdentifier(account bitbucket.Account) string {
	if account.Uuid != "" {
		return account.Uuid
	}

	return account.Username
}

// flattenBranchRestrictionUsers reports the identifier the API returns for
// each user, preferring uuid since usernames are deprecated and no longer
// returned by the API.
func flattenBranchRestrictionUsers(users []bitbucket.Account) []string {
	flattened := make([]string, 0, len(users))

	for _, user := range users {
		if id := accountIdentifier(user); id != "" {
			flattened = append(flattened, id)
		}
	}

	return flattened
}

// groupSlugIdentifier returns the workspace slug for a group, if the API
// provided one — either on the group.Workspace object, or (observed on some
// API responses) as group.Owner.Username.
func groupSlugIdentifier(group bitbucket.Group) string {
	if group.Workspace != nil && group.Workspace.Slug != "" {
		return group.Workspace.Slug
	}
	if group.Owner != nil && group.Owner.Username != "" {
		return group.Owner.Username
	}
	return ""
}

// groupUUIDIdentifier returns the workspace UUID for a group, if the API
// provided one.
func groupUUIDIdentifier(group bitbucket.Group) string {
	if group.Workspace != nil && group.Workspace.Uuid != "" {
		return group.Workspace.Uuid
	}
	if group.Owner != nil && group.Owner.Uuid != "" {
		return group.Owner.Uuid
	}
	return ""
}

// groupOwnerIdentifier resolves the default identifier to report for a
// group's owner when there's no matching prior state to preserve (see
// flattenBranchRestrictionGroups): the workspace slug, since that's the form
// the docs call preferred and what most existing configs use. Falls back to
// the UUID if the API didn't return a slug. Returns "" if neither is
// available (caller must drop the group).
func groupOwnerIdentifier(group bitbucket.Group) string {
	if slug := groupSlugIdentifier(group); slug != "" {
		return slug
	}
	return groupUUIDIdentifier(group)
}

// flattenBranchRestrictionGroups reports the owner/slug pairs the API returns
// for groups, since brRes.Groups is a slice of structs that doesn't match the
// groups TypeSet's Resource{owner, slug} shape.
//
// prevState is the groups TypeSet from state before this Read call (i.e.
// d.Get("groups").(*schema.Set) read before d.Set("groups", ...) overwrites it).
//
// The API returns both a workspace slug and a workspace UUID for the group
// owner. Either form is valid in config, but whichever form is already in
// state must round-trip stably — reporting the other (if equivalent) form
// produces a perpetual TypeSet hash mismatch even though both refer to the
// same workspace. So: if the previous owner matches either the slug or the
// UUID the API returned for this group (case-insensitive), it is kept as-is;
// otherwise the slug is used, since that's the default/preferred form (see
// groupOwnerIdentifier).
//
// Groups whose owner cannot be resolved are silently dropped — writing owner:""
// would guarantee a persistent diff because owner is Required in the schema and
// no real HCL config can supply an empty string there.
func flattenBranchRestrictionGroups(groups []bitbucket.Group, prevState *schema.Set) []interface{} {
	// Build a group slug → owner lookup from the previous state so we can
	// preserve whichever identifier form (UUID or slug) is already there.
	prevOwnerBySlug := make(map[string]string)
	if prevState != nil {
		for _, raw := range prevState.List() {
			m := raw.(map[string]interface{})
			if slug, ok := m["slug"].(string); ok && slug != "" {
				if owner, ok := m["owner"].(string); ok && owner != "" {
					prevOwnerBySlug[slug] = owner
				}
			}
		}
	}

	flattened := make([]interface{}, 0, len(groups))

	for _, group := range groups {
		owner := groupOwnerIdentifier(group)
		if owner == "" {
			continue
		}

		if prev, ok := prevOwnerBySlug[group.Slug]; ok {
			slug := groupSlugIdentifier(group)
			uuid := groupUUIDIdentifier(group)
			if strings.EqualFold(prev, slug) || strings.EqualFold(prev, uuid) {
				owner = prev
			}
		}

		flattened = append(flattened, map[string]interface{}{
			"owner": owner,
			"slug":  group.Slug,
		})
	}

	return flattened
}
