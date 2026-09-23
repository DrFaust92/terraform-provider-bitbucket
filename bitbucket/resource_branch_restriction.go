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

	// Warn when prevState contains legacy username-format users but the API
	// now returns only UUIDs (Bitbucket deprecated and removed username from
	// account responses as part of its GDPR changes). Keeping a username in
	// config produces a perpetual diff on every plan/apply because the API
	// accepts usernames on write but never echoes them back on read.
	// The warning lists each affected user's UUID and display name so the
	// operator can update their config.  The diff disappears permanently once
	// config is updated to use UUIDs. See: github.com/DrFaust92/terraform-provider-bitbucket/issues/173
	prevUsers := d.Get("users").(*schema.Set)
	if prevUsers.Len() > 0 && len(brRes.Users) > 0 {
		var staleUsernames []string
		for _, raw := range prevUsers.List() {
			if s := raw.(string); !bitbucketUUIDPattern.MatchString(s) {
				staleUsernames = append(staleUsernames, s)
			}
		}
		if len(staleUsernames) > 0 {
			var lines []string
			for _, u := range brRes.Users {
				if u.Uuid != "" {
					name := u.DisplayName
					if name == "" {
						name = u.Uuid
					}
					lines = append(lines, fmt.Sprintf("  • %s (%s)", u.Uuid, name))
				}
			}
			if len(lines) > 0 {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Warning,
					Summary:  "branch_restriction: users migrating from legacy username to UUID",
					Detail: fmt.Sprintf(
						"This resource has legacy username(s) in state (%s), but the "+
							"Bitbucket API no longer returns usernames (deprecated as part of "+
							"Bitbucket's GDPR changes). Terraform will update state to use UUIDs, "+
							"producing a perpetual diff on every plan/apply until the config is "+
							"updated to use UUIDs.\n\nUsers currently on this restriction:\n%s\n\n"+
							"Update your config to use the UUIDs above. As a temporary escape "+
							"hatch, add `lifecycle { ignore_changes = [users] }` to suppress the "+
							"diff without migrating.",
						strings.Join(staleUsernames, ", "),
						strings.Join(lines, "\n"),
					),
				})
			}
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

// groupOwnerIdentifier resolves the workspace identifier for a group returned
// by the Bitbucket branch-restrictions API.
//
// The API currently returns the workspace slug in group.Owner.Username with
// group.Owner.Uuid empty. However, the response may also include a
// group.Workspace object that carries the workspace UUID — checked first so
// users who configure groups.owner with a UUID see a stable round-trip if the
// API provides it.
//
// Priority: group.Workspace.Uuid → group.Owner.Uuid → group.Workspace.Slug → group.Owner.Username
// Returns "" if no identifier can be resolved (caller must drop the group).
func groupOwnerIdentifier(group bitbucket.Group) string {
	if group.Workspace != nil && group.Workspace.Uuid != "" {
		return group.Workspace.Uuid
	}
	if group.Owner != nil && group.Owner.Uuid != "" {
		return group.Owner.Uuid
	}
	if group.Workspace != nil && group.Workspace.Slug != "" {
		return group.Workspace.Slug
	}
	if group.Owner != nil && group.Owner.Username != "" {
		return group.Owner.Username
	}
	return ""
}

// flattenBranchRestrictionGroups reports the owner/slug pairs the API returns
// for groups, since brRes.Groups is a slice of structs that doesn't match the
// groups TypeSet's Resource{owner, slug} shape.
//
// prevState is the groups TypeSet from state before this Read call (i.e.
// d.Get("groups").(*schema.Set) read before d.Set("groups", ...) overwrites it).
// When the API returns a slug owner and prevState has a UUID owner for the same
// group slug, the UUID is preserved — preventing a perpetual diff for users who
// configure groups.owner as a UUID while the API returns the workspace slug.
//
// Groups whose owner cannot be resolved are silently dropped — writing owner:""
// would guarantee a persistent diff because owner is Required in the schema and
// no real HCL config can supply an empty string there.
// See groupOwnerIdentifier for the resolution priority.
func flattenBranchRestrictionGroups(groups []bitbucket.Group, prevState *schema.Set) []interface{} {
	// Build a slug→owner lookup from the previous state so we can preserve
	// UUID owners that the API no longer returns.
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

		// If the API returned a non-UUID owner (i.e. workspace slug) but the
		// prior state had a UUID for the same group slug, keep the UUID.
		// This prevents a perpetual diff when the user's config uses a UUID.
		if !bitbucketUUIDPattern.MatchString(owner) {
			if prev, ok := prevOwnerBySlug[group.Slug]; ok && bitbucketUUIDPattern.MatchString(prev) {
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
