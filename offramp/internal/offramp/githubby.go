// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"fmt"
	"strings"
)

// git-hubby (github.com/Interhyp/git-hubby, Apache-2.0) is the GitHub
// operator the converter treats as GitHub's provider: its Repository and
// Organization take github_repository and github_organization_settings,
// and every field it has wins over github-krm-extensions' kinds, which fill
// only what it lacks. So github_branch_default folds into
// Repository.defaultBranch, and the organization's Actions permissions into
// Organization.actionsSettings, when they are in the same state.
//
// git-hubby is authoritative where Terraform is not: what its spec leaves
// out it resets to its own defaults, and its lists (topics, webhooks,
// deploy keys, autolinks, rulesets, runner groups) are the whole set, so
// anything on GitHub not listed is removed. Every converted object
// therefore spells out each field from the state, and the lists Terraform
// did not manage get a note to read GitHub before applying.

// Group and version of git-hubby's kinds.
const githubbyAPI = "github.interhyp.de/v1alpha1"

// isGitHubby says whether an apiVersion is git-hubby's.
func isGitHubby(apiVersion string) bool {
	g, _, _ := strings.Cut(apiVersion, "/")
	return g == "github.interhyp.de"
}

// githubbyObject is one git-hubby object. It carries no per-object deletion
// policy: what deleting a Repository does is the operator's
// REPOSITORY_FINALIZER_MODE (githubbyDeletionNote).
func githubbyObject(r *TFResource, kind, name string, spec map[string]any) *Object {
	return &Object{
		Source: r.Address, Call: r.Call, TFType: r.Type, Kind: kind, Deps: r.Deps,
		Obj: map[string]any{
			"apiVersion": githubbyAPI,
			"kind":       kind,
			"metadata":   map[string]any{"name": sanitize(name), "annotations": map[string]any{}},
			"spec":       spec,
		},
	}
}

// githubbyOrgName is the Organization object's name, which every Repository
// references.
func githubbyOrgName(owner string) string { return sanitize(owner) }

// convertGHRepository: github_repository to git-hubby's Repository. An
// existing repository of the same name is adopted (git-hubby reads it by
// name and updates it), so no adopt annotation is needed.
func convertGHRepository(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	name := str(a["name"])
	var f []Finding
	branch := c.defaultBranch[name]
	if branch == "" {
		branch = str(a["default_branch"])
	}
	strategies := []any{}
	for _, s := range []struct{ attr, typ string }{
		{"allow_merge_commit", "merge"}, {"allow_rebase_merge", "rebase"}, {"allow_squash_merge", "squash"},
	} {
		if b, _ := a[s.attr].(bool); b {
			strategies = append(strategies, map[string]any{"type": s.typ})
		}
	}
	topics := []any{}
	for _, t := range list(a["topics"]) {
		topics = append(topics, map[string]any{"name": str(t)})
	}
	about := map[string]any{"description": str(a["description"]), "topics": topics}
	if h := str(a["homepage_url"]); h != "" {
		if !strings.HasPrefix(h, "http://") && !strings.HasPrefix(h, "https://") {
			f = append(f, Finding{r.Address, r.Type, "note", "homepage_url " + h + " has no http(s) scheme, which git-hubby's website requires: not converted"})
		} else {
			about["website"] = h
		}
	}
	spec := map[string]any{
		"name":                name,
		"organizationRef":     map[string]any{"name": githubbyOrgName(c.owner)},
		"visibility":          str(a["visibility"]),
		"hasIssues":           boolOf(a["has_issues"]),
		"hasProjects":         boolOf(a["has_projects"]),
		"hasWiki":             boolOf(a["has_wiki"]),
		"hasDownloads":        boolOf(a["has_downloads"]),
		"isTemplate":          boolOf(a["is_template"]),
		"archived":            boolOf(a["archived"]),
		"deleteBranchOnMerge": boolOf(a["delete_branch_on_merge"]),
		"mergeCommitTitle":    str(a["merge_commit_title"]),
		"mergeCommitMessage":  str(a["merge_commit_message"]),
		"defaultBranch":       branch,
		"about":               about,
	}
	if len(strategies) > 0 {
		spec["allowedMergeStrategies"] = strategies
	}
	for k, v := range spec {
		if v == nil || v == "" {
			delete(spec, k)
		}
	}
	if u := githubbyUnmanaged(a); len(u) > 0 {
		f = append(f, Finding{r.Address, r.Type, "note", "git-hubby has no field for " + strings.Join(u, ", ") + ": GitHub keeps the current values, managed by nothing after the offramp"})
	}
	return []*Object{githubbyObject(r, "Repository", name, spec)}, f
}

// githubbyUnmanaged lists the repository settings Terraform holds, set away
// from GitHub's defaults, that git-hubby's Repository has no field for.
func githubbyUnmanaged(a map[string]any) []string {
	var out []string
	for _, s := range []struct {
		attr string
		def  any
	}{
		{"allow_auto_merge", false}, {"allow_update_branch", false}, {"has_discussions", false},
		{"web_commit_signoff_required", false}, {"vulnerability_alerts", false},
		{"squash_merge_commit_title", "COMMIT_OR_PR_TITLE"}, {"squash_merge_commit_message", "COMMIT_MESSAGES"},
	} {
		if v, ok := a[s.attr]; ok && v != nil && v != s.def {
			out = append(out, s.attr)
		}
	}
	for _, blk := range []string{"pages", "template"} {
		if len(list(a[blk])) > 0 {
			out = append(out, blk)
		}
	}
	if securityEnabled(a["security_and_analysis"]) {
		out = append(out, "security_and_analysis")
	}
	return out
}

// securityEnabled says whether any security_and_analysis feature is on.
func securityEnabled(v any) bool {
	for _, b := range list(v) {
		m, _ := b.(map[string]any)
		for _, feat := range m {
			for _, x := range list(feat) {
				if fm, _ := x.(map[string]any); str(fm["status"]) == "enabled" {
					return true
				}
			}
		}
	}
	return false
}

// convertGHOrganization: github_organization_settings to git-hubby's
// Organization, with the organization's Actions permissions from the same
// state. Two inputs the state does not hold are required: the plan, which
// decides what git-hubby reconciles, and the GitHub App installation it
// authenticates as (Options.GitHubPlan and GitHubAppInstallation).
func convertGHOrganization(r *TFResource, c *githubContext) ([]*Object, []Finding) {
	a := r.Attrs
	var missing []string
	if c.plan == "" {
		missing = append(missing, "--github-plan (free, team or enterprise; git-hubby's default is enterprise)")
	}
	if c.installation == 0 {
		missing = append(missing, "--github-app-installation (the git-hubby GitHub App's installation ID)")
	}
	if len(missing) > 0 {
		return nil, []Finding{{r.Address, r.Type, "gap", "git-hubby's Organization needs " + strings.Join(missing, " and ") + ", which the state does not hold: run the converter with them"}}
	}
	f := []Finding{{r.Address, r.Type, "note", "Terraform did not manage the organization's artifact and log retention; git-hubby sets it (400 days unless given), so it is written as 90, GitHub's default: compare GET /orgs/{org}/actions/permissions/artifact-and-log-retention before applying"},
		{r.Address, r.Type, "note", "git-hubby deletes runner groups it does not list; only Default is listed: list any others (GET /orgs/{org}/actions/runner-groups) before applying"}}
	priv := map[string]any{
		"defaultRepositoryPermission":         str(a["default_repository_permission"]),
		"membersCanCreatePublicRepositories":  boolOf(a["members_can_create_public_repositories"]),
		"membersCanCreatePrivateRepositories": boolOf(a["members_can_create_private_repositories"]),
		"membersCanCreatePages":               boolOf(a["members_can_create_pages"]),
		"membersCanCreatePublicPages":         boolOf(a["members_can_create_public_pages"]),
		"membersCanCreatePrivatePages":        boolOf(a["members_can_create_private_pages"]),
		"membersCanForkPrivateRepositories":   boolOf(a["members_can_fork_private_repositories"]),
	}
	if c.plan == "enterprise" {
		priv["membersCanCreateInternalRepositories"] = boolOf(a["members_can_create_internal_repositories"])
	}
	for k, v := range priv {
		if v == nil || v == "" {
			delete(priv, k)
		}
	}
	actions, af := githubbyActions(r, c)
	f = append(f, af...)
	spec := map[string]any{
		"login":                      c.owner,
		"name":                       str(a["name"]),
		"description":                str(a["description"]),
		"plan":                       c.plan,
		"githubAppConfig":            map[string]any{"installationId": c.installation, "credentialsSecretName": "git-hubby-app"},
		"actionsSettings":            actions,
		"codeSecurityConfigurations": []any{},
		"memberPrivileges":           priv,
	}
	if v := str(a["location"]); v != "" {
		spec["location"] = v
	}
	if v := str(a["blog"]); v != "" {
		spec["website"] = v
	}
	var u []string
	for _, k := range sortedKeys(a) {
		if !githubbyOrgCovered[k] && a[k] != nil && a[k] != "" && a[k] != false {
			u = append(u, k)
		}
	}
	if len(u) > 0 {
		f = append(f, Finding{r.Address, r.Type, "note", "git-hubby has no field for " + strings.Join(u, ", ") + ": GitHub keeps the current values, managed by nothing after the offramp"})
	}
	return []*Object{githubbyObject(r, "Organization", c.owner, spec)}, f
}

// githubbyOrgCovered: github_organization_settings attributes the
// Organization carries, or that are not settings.
var githubbyOrgCovered = map[string]bool{
	"id": true, "name": true, "description": true, "location": true, "blog": true,
	"default_repository_permission": true, "members_can_create_public_repositories": true,
	"members_can_create_private_repositories": true, "members_can_create_internal_repositories": true,
	"members_can_create_pages": true, "members_can_create_public_pages": true,
	"members_can_create_private_pages": true, "members_can_fork_private_repositories": true,
	// The aggregate of the public, private and internal settings.
	"members_can_create_repositories": true,
}

// githubbyActions: the organization's Actions settings from
// github_actions_organization_permissions and _workflow_permissions. With
// none, git-hubby turns Actions off for the organization, so the
// Organization says what GitHub does by default: enabled for all.
func githubbyActions(r *TFResource, c *githubContext) (map[string]any, []Finding) {
	var f []Finding
	s := map[string]any{
		"artifactAndLogRetentionDays": 90,
		"runnerGroups":                []any{map[string]any{"name": "Default", "visibility": "all"}},
	}
	if p := c.orgActions; p != nil {
		s["enabledRepositories"] = str(p["enabled_repositories"])
		s["allowedActions"] = str(p["allowed_actions"])
		s["shaPinningRequired"] = boolOf(p["sha_pinning_required"])
		if cfg := list(p["allowed_actions_config"]); len(cfg) > 0 {
			m, _ := cfg[0].(map[string]any)
			sel := map[string]any{
				"githubOwnedAllowed": boolOf(m["github_owned_allowed"]),
				"verifiedAllowed":    boolOf(m["verified_allowed"]),
			}
			if pats := list(m["patterns_allowed"]); len(pats) > 0 {
				sel["patternsAllowed"] = pats
			}
			s["selectedAllowedActions"] = sel
		}
		if len(list(p["enabled_repositories_config"])) > 0 {
			f = append(f, Finding{r.Address, "github_actions_organization_permissions", "note", "the selected repositories are each Repository's actionsEnabled in git-hubby: set it on the repositories enabled_repositories_config lists"})
		}
	} else {
		s["enabledRepositories"] = "all"
		s["allowedActions"] = "all"
		f = append(f, Finding{r.Address, r.Type, "note", "no github_actions_organization_permissions in the state: git-hubby turns Actions off without enabledRepositories, so it is written as GitHub's default (all repositories, all actions)"})
	}
	if w := c.orgWorkflow; w != nil {
		s["defaultWorkflowPermissions"] = str(w["default_workflow_permissions"])
		s["canApprovePullRequestReviews"] = boolOf(w["can_approve_pull_request_reviews"])
	}
	for k, v := range s {
		if v == nil || v == "" {
			delete(s, k)
		}
	}
	return s, f
}

// githubbyFolded: a resource whose fields a git-hubby object in the same
// state carries. It converts to nothing, with a note naming where it went.
func githubbyFolded(into string) githubConverter {
	return func(r *TFResource, _ *githubContext) ([]*Object, []Finding) {
		return nil, []Finding{{r.Address, r.Type, "note", "carried by git-hubby's " + into}}
	}
}

// githubbyDeletionNote: git-hubby decides what deleting a Repository does
// operator-wide, not per object; with --abandon it must stay at ignore.
func githubbyDeletionNote(objs []*Object, opt Options) []Finding {
	n := 0
	for _, o := range objs {
		if isGitHubby(str(o.Obj["apiVersion"])) && o.Kind == "Repository" {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	mode := "ignore (its default), so deleting a Repository object leaves the repository"
	if opt.DeletionPolicy == "delete" {
		mode = "archive or delete for deleting a Repository object to remove the repository; its default, ignore, leaves it"
	}
	return []Finding{{"", "github_repository", "note", fmt.Sprintf("%d git-hubby Repositories: git-hubby has no per-object deletion policy; run it with REPOSITORY_FINALIZER_MODE=%s", n, mode)}}
}
