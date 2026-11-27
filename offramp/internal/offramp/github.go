// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"encoding/base64"
	"strings"
)

// GitHub: Terraform's integrations/github provider to github-krm-extensions'
// kinds (krm-rest-runtime). Spec fields keep GitHub's own names (the REST
// API's snake_case, GraphQL's camelCase for BranchProtection). Every object
// carries krm-rest-runtime's annotations (its README, "Conventions"):
// deletion-policy abandon (the default for an import), and, for a CRUD kind,
// adopt-id: the ID status.id would hold, so the first reconcile reads the
// resource instead of creating a second one. Settings and upsert kinds,
// whose path is all spec fields, are read before they are written and need
// no adopt-id.

// krm-rest-runtime annotations.
const (
	AnnRESTAdoptID        = "krm.hfl.dev/adopt-id"
	AnnRESTDeletionPolicy = "krm.hfl.dev/deletion-policy"
)

// isREST says whether an apiVersion is a krm-rest-runtime vendor's.
func isREST(apiVersion string) bool {
	g, _, _ := strings.Cut(apiVersion, "/")
	return strings.HasSuffix(g, ".github.hfl.dev")
}

// githubContext is what the GitHub converters need from the whole state:
// the owner (the provider's, which no resource holds), read from a
// repository's full_name, or an organization-level resource's ID.
type githubContext struct {
	owner string
	// From the rest of the state, folded into git-hubby's objects
	// (githubby.go): each repository's default branch, and the
	// organization's Actions permissions when its settings are here too.
	defaultBranch           map[string]string
	orgActions, orgWorkflow map[string]any
	hasOrg, hasRepo         map[string]bool
	// From Options: whether git-hubby takes the repository and the
	// organization's settings, and what its Organization needs that no
	// state holds.
	githubby     bool
	plan         string
	installation int64
}

func newGitHubContext(rs []*TFResource, opt Options) *githubContext {
	c := &githubContext{defaultBranch: map[string]string{}, hasOrg: map[string]bool{}, hasRepo: map[string]bool{},
		githubby: opt.GitHubOperator == "git-hubby", plan: opt.GitHubPlan, installation: opt.GitHubAppInstallation}
	for _, r := range rs {
		switch r.Type {
		case "github_organization_settings":
			c.hasOrg["settings"] = true
		case "github_branch_default":
			c.defaultBranch[str(r.Attrs["repository"])] = str(r.Attrs["branch"])
		}
		if r.Type == "github_repository" {
			c.hasRepo[str(r.Attrs["name"])] = true
		}
		switch r.Type {
		case "github_repository":
			if o, _, ok := strings.Cut(str(r.Attrs["full_name"]), "/"); ok && c.owner == "" {
				c.owner = o
			}
		case "github_actions_organization_permissions", "github_actions_organization_workflow_permissions":
			if c.owner == "" {
				c.owner = str(r.Attrs["id"])
			}
			if r.Type == "github_actions_organization_permissions" {
				c.orgActions = r.Attrs
			} else {
				c.orgWorkflow = r.Attrs
			}
		}
	}
	return c
}

type githubConverter func(r *TFResource, c *githubContext) ([]*Object, []Finding)

var githubConverters = map[string]githubConverter{
	"github_actions_organization_permissions":          convertGHOrgActionsPermissions,
	"github_actions_organization_workflow_permissions": convertGHOrgWorkflowPermissions,
	"github_actions_repository_permissions":            convertGHRepoActionsPermissions,
	"github_workflow_repository_permissions":           convertGHRepoWorkflowPermissions,
	"github_actions_variable":                          convertGHActionsVariable,
	"github_branch_default":                            convertGHBranchDefault,
	"github_branch_protection":                         convertGHBranchProtection,
	"github_repository_file":                           convertGHRepositoryFile,
	"github_repository":                                byOperator(convertGHRepositoryREST, convertGHRepository),
	"github_organization_settings":                     byOperator(convertGHOrgSettings, convertGHOrganization),
}

// byOperator: github-krm-extensions' converter, or git-hubby's with
// --github-operator git-hubby.
func byOperator(ext, hubby githubConverter) githubConverter {
	return func(r *TFResource, c *githubContext) ([]*Object, []Finding) {
		if c.githubby {
			return hubby(r, c)
		}
		return ext(r, c)
	}
}

// githubbyFolds: our kinds whose fields git-hubby's objects carry when the
// object they fold into is in the same state.
func githubbyFolds(r *TFResource, c *githubContext) githubConverter {
	if !c.githubby {
		return nil
	}
	switch r.Type {
	case "github_branch_default":
		if c.hasRepo[str(r.Attrs["repository"])] {
			return githubbyFolded("Repository (defaultBranch)")
		}
	case "github_actions_organization_permissions", "github_actions_organization_workflow_permissions":
		if c.hasOrg["settings"] {
			return githubbyFolded("Organization (actionsSettings)")
		}
	}
	return nil
}

func convertGitHub(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	if fold := githubbyFolds(r, c); fold != nil {
		return fold(r, c)
	}
	f, ok := githubConverters[r.Type]
	if !ok {
		return nil, []Finding{{r.Address, r.Type, "gap", "no GitHub mapping in the converter yet (github-krm-extensions' README names the kind): keep it in Terraform, or add it to github.go"}}
	}
	if c.owner == "" && needsOwner[r.Type] {
		return nil, []Finding{{r.Address, r.Type, "gap", "the repository's owner is the provider's and the state names none (no github_repository full_name): convert it with the owner set by hand"}}
	}
	return f(r, c)
}

var needsOwner = map[string]bool{
	"github_actions_repository_permissions": true, "github_workflow_repository_permissions": true,
	"github_actions_variable": true, "github_branch_default": true, "github_repository_file": true,
	"github_repository": true, "github_organization_settings": true,
}

// convertGHRepositoryREST: github_repository to Repository, a CRUD kind
// whose ID is the repository's name, so an existing repository is adopted.
// Terraform's resource makes other calls too, each another kind here:
// topics (RepositoryTopics) and vulnerability alerts
// (RepositoryVulnerabilityAlerts). Fields only a create takes (auto_init,
// templates, has_downloads) are left out: an adopted repository is never
// created. Its default branch is github_branch_default's (BranchDefault).
func convertGHRepositoryREST(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	name := str(a["name"])
	spec := map[string]any{
		"owner": c.owner, "name": name,
		"description": str(a["description"]),
		"homepage":    str(a["homepage_url"]),
		"visibility":  str(a["visibility"]),
	}
	for _, k := range []string{
		"has_issues", "has_projects", "has_wiki", "is_template", "archived",
		"allow_merge_commit", "allow_squash_merge", "allow_rebase_merge", "allow_auto_merge",
		"allow_update_branch", "allow_forking", "delete_branch_on_merge", "web_commit_signoff_required",
	} {
		spec[k] = boolOf(a[k])
	}
	for _, k := range []string{"merge_commit_title", "merge_commit_message", "squash_merge_commit_title", "squash_merge_commit_message"} {
		spec[k] = str(a[k])
	}
	if sa := securityAndAnalysis(a["security_and_analysis"]); len(sa) > 0 {
		spec["security_and_analysis"] = sa
	}
	objs := []*Object{restObject(r, "repository", "Repository", name, spec, name)}
	var f []Finding
	if topics := list(a["topics"]); len(topics) > 0 {
		objs = append(objs, restObject(r, "repository", "RepositoryTopics", name+"-topics",
			map[string]any{"owner": c.owner, "repo": name, "names": topics}, ""))
	}
	if b, _ := a["vulnerability_alerts"].(bool); b {
		objs = append(objs, restObject(r, "repository", "RepositoryVulnerabilityAlerts", name+"-vulnerability-alerts",
			map[string]any{"owner": c.owner, "repo": name}, ""))
	}
	if b, _ := a["has_discussions"].(bool); b {
		f = append(f, Finding{r.Address, r.Type, "note", "has_discussions is not in GitHub's REST update body: discussions stay as GitHub has them"})
	}
	if len(list(a["pages"])) > 0 {
		f = append(f, Finding{r.Address, r.Type, "gap", "the repository's Pages site: RepositoryPagesSite cannot set a source yet (github-krm-extensions' live.yaml): keep it in Terraform"})
	}
	if b, _ := a["archive_on_destroy"].(bool); b {
		f = append(f, Finding{r.Address, r.Type, "note", "archive_on_destroy: Terraform archives the repository on destroy; deleting a Repository object with deletion-policy delete deletes it"})
	}
	return objs, f
}

// securityAndAnalysis: Terraform's security_and_analysis block (each
// feature a one-element list of {status}) as the API's object.
func securityAndAnalysis(v any) map[string]any {
	out := map[string]any{}
	for _, b := range list(v) {
		m, _ := b.(map[string]any)
		for feat, x := range m {
			if l := list(x); len(l) > 0 {
				if fm, _ := l[0].(map[string]any); str(fm["status"]) != "" {
					out[feat] = map[string]any{"status": str(fm["status"])}
				}
			}
		}
	}
	return out
}

// convertGHOrgSettings: github_organization_settings to
// OrganizationSettings, a settings kind on the organization (never
// deleted).
func convertGHOrgSettings(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{"org": c.owner}
	for _, k := range []string{"billing_email", "blog", "company", "description", "email", "location", "name",
		"twitter_username", "default_repository_permission", "secret_scanning_push_protection_custom_link"} {
		spec[k] = str(a[k])
	}
	for _, k := range []string{"has_organization_projects", "has_repository_projects", "members_can_create_repositories",
		"members_can_create_public_repositories", "members_can_create_private_repositories",
		"members_can_create_pages", "members_can_create_public_pages", "members_can_create_private_pages",
		"members_can_fork_private_repositories", "web_commit_signoff_required",
		"secret_scanning_push_protection_custom_link_enabled"} {
		spec[k] = boolOf(a[k])
	}
	// Internal repositories exist only in an enterprise: sent only when on.
	if b, _ := a["members_can_create_internal_repositories"].(bool); b {
		spec["members_can_create_internal_repositories"] = true
	}
	var f []Finding
	var dep []string
	for _, k := range sortedKeys(a) {
		if strings.HasSuffix(k, "_enabled_for_new_repositories") {
			if b, _ := a[k].(bool); b {
				dep = append(dep, k)
			}
		}
	}
	if len(dep) > 0 {
		f = append(f, Finding{r.Address, r.Type, "note", "GitHub has deprecated " + strings.Join(dep, ", ") + " for code security configurations: not converted"})
	}
	return []*Object{restObject(r, "organization", "OrganizationSettings", c.owner, spec, "")}, f
}

// restObject is one github-krm-extensions object; adoptID is set for a CRUD
// kind.
func restObject(r *TFResource, group, kind, name string, spec map[string]any, adoptID string) *Object {
	for k, v := range spec {
		if v == nil || v == "" {
			delete(spec, k)
		}
	}
	ann := map[string]any{}
	if adoptID != "" {
		ann[AnnRESTAdoptID] = adoptID
	}
	return &Object{
		Source: r.Address, Call: r.Call, TFType: r.Type, Kind: kind, Deps: r.Deps,
		Obj: map[string]any{
			"apiVersion": group + ".github.hfl.dev/v1alpha1",
			"kind":       kind,
			"metadata":   map[string]any{"name": sanitize(name), "annotations": ann},
			"spec":       spec,
		},
	}
}

// boolOf keeps a Terraform bool, which false is too (a setting turned off
// is a setting).
func boolOf(v any) any {
	if b, ok := v.(bool); ok {
		return b
	}
	return nil
}

func convertGHOrgActionsPermissions(r *TFResource, _ *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{
		"org":                  str(a["id"]),
		"enabled_repositories": str(a["enabled_repositories"]),
		"allowed_actions":      str(a["allowed_actions"]),
		"sha_pinning_required": boolOf(a["sha_pinning_required"]),
	}
	var f []Finding
	if len(list(a["allowed_actions_config"])) > 0 || len(list(a["enabled_repositories_config"])) > 0 {
		f = append(f, Finding{r.Address, r.Type, "note", "allowed_actions_config and enabled_repositories_config are separate GitHub resources (selected actions, selected repositories), not converted with the permissions"})
	}
	return []*Object{restObject(r, "actions", "ActionsOrganizationPermissions", str(a["id"]), spec, "")}, f
}

func convertGHOrgWorkflowPermissions(r *TFResource, _ *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	org := str(a["organization_slug"])
	if org == "" {
		org = str(a["id"])
	}
	spec := map[string]any{
		"org":                              org,
		"default_workflow_permissions":     str(a["default_workflow_permissions"]),
		"can_approve_pull_request_reviews": boolOf(a["can_approve_pull_request_reviews"]),
	}
	return []*Object{restObject(r, "actions", "ActionsOrganizationWorkflowPermissions", org+"-workflow", spec, "")}, nil
}

func convertGHRepoActionsPermissions(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	repo := str(a["repository"])
	spec := map[string]any{
		"owner": c.owner, "repo": repo,
		"enabled":              boolOf(a["enabled"]),
		"allowed_actions":      str(a["allowed_actions"]),
		"sha_pinning_required": boolOf(a["sha_pinning_required"]),
	}
	var f []Finding
	if len(list(a["allowed_actions_config"])) > 0 {
		f = append(f, Finding{r.Address, r.Type, "note", "allowed_actions_config is a separate GitHub resource (the repository's selected actions), not converted with the permissions"})
	}
	return []*Object{restObject(r, "actions", "ActionsRepositoryPermissions", repo+"-actions", spec, "")}, f
}

func convertGHRepoWorkflowPermissions(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	repo := str(a["repository"])
	spec := map[string]any{
		"owner": c.owner, "repo": repo,
		"default_workflow_permissions":     str(a["default_workflow_permissions"]),
		"can_approve_pull_request_reviews": boolOf(a["can_approve_pull_request_reviews"]),
	}
	return []*Object{restObject(r, "workflow", "WorkflowRepositoryPermissions", repo+"-workflow", spec, "")}, nil
}

// convertGHActionsVariable: CRUD, its ID the variable's name.
func convertGHActionsVariable(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	repo, name := str(a["repository"]), str(a["variable_name"])
	spec := map[string]any{"owner": c.owner, "repo": repo, "name": name, "value": str(a["value"])}
	return []*Object{restObject(r, "actions", "ActionsVariable", repo+"-"+name, spec, name)}, nil
}

func convertGHBranchDefault(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	repo := str(a["repository"])
	spec := map[string]any{"owner": c.owner, "repo": repo, "default_branch": str(a["branch"])}
	var f []Finding
	if b, _ := a["rename"].(bool); b {
		f = append(f, Finding{r.Address, r.Type, "note", "rename renames the branch on create in Terraform; BranchDefault only points the default at a branch"})
	}
	return []*Object{restObject(r, "branch", "BranchDefault", repo+"-default-branch", spec, "")}, f
}

// convertGHBranchProtection: GraphQL's BranchProtectionRule, CRUD, its ID
// the rule's node ID (Terraform's id). Terraform's nested blocks are the
// rule's flat fields; a block present turns its feature on.
func convertGHBranchProtection(r *TFResource, _ *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{
		"repositoryId":                   str(a["repository_id"]),
		"pattern":                        str(a["pattern"]),
		"allowsDeletions":                boolOf(a["allows_deletions"]),
		"allowsForcePushes":              boolOf(a["allows_force_pushes"]),
		"isAdminEnforced":                boolOf(a["enforce_admins"]),
		"lockBranch":                     boolOf(a["lock_branch"]),
		"requiresConversationResolution": boolOf(a["require_conversation_resolution"]),
		"requiresCommitSignatures":       boolOf(a["require_signed_commits"]),
		"requiresLinearHistory":          boolOf(a["required_linear_history"]),
	}
	var f []Finding
	actors := func(v any) []any {
		var out []any
		for _, x := range list(v) {
			s := str(x)
			if strings.HasPrefix(s, "/") || strings.Contains(s, "/") {
				f = append(f, Finding{r.Address, r.Type, "note", "the actor " + s + " is a name; BranchProtection takes node IDs: replace it by the user's or team's node ID"})
			}
			out = append(out, s)
		}
		return out
	}
	if v := actors(a["force_push_bypassers"]); len(v) > 0 {
		spec["bypassForcePushActorIds"] = v
	}
	if b := list(a["required_pull_request_reviews"]); len(b) > 0 {
		m, _ := b[0].(map[string]any)
		spec["requiresApprovingReviews"] = true
		spec["dismissesStaleReviews"] = boolOf(m["dismiss_stale_reviews"])
		spec["requiresCodeOwnerReviews"] = boolOf(m["require_code_owner_reviews"])
		spec["requireLastPushApproval"] = boolOf(m["require_last_push_approval"])
		spec["restrictsReviewDismissals"] = boolOf(m["restrict_dismissals"])
		if n, ok := m["required_approving_review_count"].(float64); ok {
			spec["requiredApprovingReviewCount"] = int64(n)
		}
		if v := actors(m["dismissal_restrictions"]); len(v) > 0 {
			spec["reviewDismissalActorIds"] = v
		}
		if v := actors(m["pull_request_bypassers"]); len(v) > 0 {
			spec["bypassPullRequestActorIds"] = v
		}
	}
	if b := list(a["required_status_checks"]); len(b) > 0 {
		m, _ := b[0].(map[string]any)
		spec["requiresStatusChecks"] = true
		spec["requiresStrictStatusChecks"] = boolOf(m["strict"])
		if l := list(m["contexts"]); len(l) > 0 {
			spec["requiredStatusCheckContexts"] = l
		}
	}
	if b := list(a["restrict_pushes"]); len(b) > 0 {
		m, _ := b[0].(map[string]any)
		spec["restrictsPushes"] = true
		spec["blocksCreations"] = boolOf(m["blocks_creations"])
		if v := actors(m["push_allowances"]); len(v) > 0 {
			spec["pushActorIds"] = v
		}
	}
	for k, v := range spec {
		if v == nil {
			delete(spec, k)
		}
	}
	return []*Object{restObject(r, "branch", "BranchProtection", str(a["repository_id"])+"-"+str(a["pattern"]), spec, str(a["id"]))}, f
}

// convertGHRepositoryFile: the contents API takes the file base64-encoded;
// Terraform holds it as text. sha is the blob GitHub answers, carried on
// each write by the kind itself, so it is not written here.
func convertGHRepositoryFile(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	repo, path := str(a["repository"]), str(a["file"])
	spec := map[string]any{
		"owner": c.owner, "repo": repo, "path": path,
		"branch":  str(a["branch"]),
		"message": str(a["commit_message"]),
		"content": base64.StdEncoding.EncodeToString([]byte(str(a["content"]))),
	}
	if n, e := str(a["commit_author"]), str(a["commit_email"]); n != "" || e != "" {
		spec["committer"] = map[string]any{"name": n, "email": e}
	}
	return []*Object{restObject(r, "repository", "RepositoryFile", repo+"-"+path, spec, "")}, nil
}
