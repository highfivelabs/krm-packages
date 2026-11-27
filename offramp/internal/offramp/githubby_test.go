// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// githubbyState: an organization's settings, its Actions permissions, and
// two repositories (one public with a homepage and topics, one private),
// in the shapes integrations/github keeps them.
func githubbyState() *State {
	return &State{Version: 4, Resources: []StateResource{
		ghResource("github_organization_settings", "org", map[string]any{
			"id": "1", "name": "Acme", "description": "Things", "location": "Earth", "blog": "https://acme.example",
			"billing_email": "billing@acme.example", "default_repository_permission": "read",
			"members_can_create_public_repositories": true, "members_can_create_private_repositories": false,
			"members_can_create_internal_repositories": false, "members_can_create_pages": true,
			"members_can_fork_private_repositories": false, "has_organization_projects": true,
		}),
		ghResource("github_actions_organization_permissions", "actions", map[string]any{
			"id": "acme", "enabled_repositories": "all", "allowed_actions": "selected", "sha_pinning_required": false,
			"allowed_actions_config": []any{map[string]any{"github_owned_allowed": true, "verified_allowed": false, "patterns_allowed": []any{"acme/*"}}},
		}),
		ghResource("github_actions_organization_workflow_permissions", "wf", map[string]any{
			"id": "acme", "organization_slug": "acme", "default_workflow_permissions": "read", "can_approve_pull_request_reviews": false,
		}),
		ghResource("github_repository", "site", map[string]any{
			"name": "site", "full_name": "acme/site", "visibility": "public", "description": "The site",
			"homepage_url": "https://acme.example", "topics": []any{"web", "docs"},
			"has_issues": false, "has_projects": false, "has_wiki": false, "has_downloads": false,
			"is_template": false, "archived": false, "delete_branch_on_merge": false,
			"allow_merge_commit": true, "allow_squash_merge": true, "allow_rebase_merge": false,
			"merge_commit_title": "MERGE_MESSAGE", "merge_commit_message": "PR_TITLE",
			"squash_merge_commit_title": "COMMIT_OR_PR_TITLE", "squash_merge_commit_message": "COMMIT_MESSAGES",
			"vulnerability_alerts": true, "default_branch": "main",
		}),
		ghResource("github_branch_default", "site", map[string]any{"repository": "site", "branch": "trunk"}),
		ghResource("github_repository", "api", map[string]any{
			"name": "api", "full_name": "acme/api", "visibility": "private", "homepage_url": "acme.example",
			"has_issues": true, "allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": false,
			"allow_auto_merge": true, "default_branch": "main",
		}),
	}}
}

func githubbyCRDs(t *testing.T) map[string]*CRD {
	t.Helper()
	out := map[string]*CRD{}
	files, _ := filepath.Glob("testdata/githubby/*.yaml")
	if len(files) != 2 {
		t.Fatalf("want git-hubby's two CRDs in testdata/githubby, got %v", files)
	}
	for _, f := range files {
		c, err := loadCRD(f)
		if err != nil || c == nil {
			t.Fatalf("%s: %v", f, err)
		}
		out[c.APIVersion()+"/"+c.Kind] = c
	}
	return out
}

var githubbyOpt = Options{GitHubOperator: "git-hubby", GitHubPlan: "free", GitHubAppInstallation: 42}

// TestGitHubbyObjectsMatchTheCRDs: every field the converter writes is in
// git-hubby's schema, and every required one is there.
func TestGitHubbyObjectsMatchTheCRDs(t *testing.T) {
	crds := githubbyCRDs(t)
	for _, opt := range []Options{githubbyOpt, {GitHubOperator: "git-hubby", GitHubPlan: "enterprise", GitHubAppInstallation: 42}} {
		res := ConvertState(githubbyState(), &KCC{}, opt)
		n := 0
		for _, o := range res.Objects {
			av := str(o.Obj["apiVersion"])
			if !isGitHubby(av) {
				continue
			}
			n++
			c, ok := crds[av+"/"+o.Kind]
			if !ok {
				t.Fatalf("%s %s: no CRD", av, o.Kind)
			}
			for _, e := range schemaErrors(o.Obj["spec"], c.Spec, "spec") {
				t.Errorf("%s %s: %s", o.Kind, o.Name(), e)
			}
		}
		if n != 3 {
			t.Errorf("plan %s: %d git-hubby objects, want 3", opt.GitHubPlan, n)
		}
	}
}

// TestGitHubbyRepository: every setting is spelled out from the state,
// false ones too, since git-hubby resets what its spec leaves out; the
// default branch comes from github_branch_default, which folds in.
func TestGitHubbyRepository(t *testing.T) {
	res := ConvertState(githubbyState(), &KCC{}, githubbyOpt)
	if g := findingsOf(res, "gap"); len(g) != 0 {
		t.Errorf("gaps: %v", g)
	}
	if got, want := kindsOf(res), []string{"Organization", "Repository", "Repository"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kinds %v, want %v (our Actions and BranchDefault kinds fold into git-hubby's)", got, want)
	}
	for _, c := range []struct {
		name string
		path []string
		want any
	}{
		{"site", []string{"spec", "defaultBranch"}, "trunk"},
		{"site", []string{"spec", "hasIssues"}, false},
		{"site", []string{"spec", "deleteBranchOnMerge"}, false},
		{"site", []string{"spec", "archived"}, false},
		{"site", []string{"spec", "visibility"}, "public"},
		{"site", []string{"spec", "allowedMergeStrategies"}, []any{map[string]any{"type": "merge"}, map[string]any{"type": "squash"}}},
		{"site", []string{"spec", "about", "website"}, "https://acme.example"},
		{"site", []string{"spec", "about", "topics"}, []any{map[string]any{"name": "web"}, map[string]any{"name": "docs"}}},
		{"site", []string{"spec", "organizationRef", "name"}, "acme"},
		{"api", []string{"spec", "defaultBranch"}, "main"},
		{"api", []string{"spec", "about", "website"}, nil},
		{"api", []string{"spec", "about", "topics"}, []any{}},
	} {
		if v := dig(objectNamed(res, "Repository", c.name), c.path...); !reflect.DeepEqual(v, c.want) {
			t.Errorf("%s %s = %#v, want %#v", c.name, strings.Join(c.path, "."), v, c.want)
		}
	}
	notes := findingsOf(res, "note")
	for _, w := range []string{
		"no field for vulnerability_alerts:",
		"no field for allow_auto_merge:",
		"homepage_url acme.example has no http(s) scheme",
		"2 git-hubby Repositories",
	} {
		if !anyContains(notes, w) {
			t.Errorf("no note saying %q in %v", w, notes)
		}
	}
}

// TestGitHubbyOrganization: the settings and the Actions permissions in one
// Organization, with the plan and App installation the state lacks.
func TestGitHubbyOrganization(t *testing.T) {
	res := ConvertState(githubbyState(), &KCC{}, githubbyOpt)
	org := objectNamed(res, "Organization", "acme")
	for _, c := range []struct {
		path []string
		want any
	}{
		{[]string{"spec", "login"}, "acme"},
		{[]string{"spec", "plan"}, "free"},
		{[]string{"spec", "website"}, "https://acme.example"},
		{[]string{"spec", "githubAppConfig", "installationId"}, int64(42)},
		{[]string{"spec", "actionsSettings", "enabledRepositories"}, "all"},
		{[]string{"spec", "actionsSettings", "allowedActions"}, "selected"},
		{[]string{"spec", "actionsSettings", "selectedAllowedActions", "patternsAllowed"}, []any{"acme/*"}},
		{[]string{"spec", "actionsSettings", "defaultWorkflowPermissions"}, "read"},
		{[]string{"spec", "actionsSettings", "canApprovePullRequestReviews"}, false},
		{[]string{"spec", "actionsSettings", "artifactAndLogRetentionDays"}, 90},
		{[]string{"spec", "memberPrivileges", "membersCanCreatePrivateRepositories"}, false},
		{[]string{"spec", "memberPrivileges", "membersCanCreateInternalRepositories"}, nil},
	} {
		if v := dig(org, c.path...); !reflect.DeepEqual(v, c.want) {
			t.Errorf("%s = %#v, want %#v", strings.Join(c.path, "."), v, c.want)
		}
	}
	notes := findingsOf(res, "note")
	for _, w := range []string{"billing_email", "has_organization_projects", "retention", "runner groups", "carried by git-hubby's Organization (actionsSettings)"} {
		if !anyContains(notes, w) {
			t.Errorf("no note saying %q in %v", w, notes)
		}
	}
	ent := ConvertState(githubbyState(), &KCC{}, Options{GitHubOperator: "git-hubby", GitHubPlan: "enterprise", GitHubAppInstallation: 42})
	if v := dig(objectNamed(ent, "Organization", "acme"), "spec", "memberPrivileges", "membersCanCreateInternalRepositories"); v != false {
		t.Errorf("enterprise: membersCanCreateInternalRepositories = %#v, want false", v)
	}
}

// TestGitHubbyOrganizationNeedsInputs: without the plan and the App
// installation the Organization is a gap naming both flags, and the
// repositories still convert.
func TestGitHubbyOrganizationNeedsInputs(t *testing.T) {
	res := ConvertState(githubbyState(), &KCC{}, Options{GitHubOperator: "git-hubby"})
	g := findingsOf(res, "gap")
	if len(g) != 1 || !strings.Contains(g[0], "--github-plan") || !strings.Contains(g[0], "--github-app-installation") {
		t.Errorf("gaps %v, want one naming both flags", g)
	}
	if got := kindsOf(res); !reflect.DeepEqual(got, []string{"Repository", "Repository"}) {
		t.Errorf("kinds %v", got)
	}
}

// TestGitHubbyFoldNeedsItsObject: a default branch or Actions permissions
// whose repository or organization settings are in another state keep our
// kinds.
func TestGitHubbyFoldNeedsItsObject(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{
		ghResource("github_repository", "a", map[string]any{"name": "a", "full_name": "acme/a"}),
		ghResource("github_branch_default", "b", map[string]any{"repository": "elsewhere", "branch": "main"}),
		ghResource("github_actions_organization_permissions", "p", map[string]any{"id": "acme", "enabled_repositories": "all"}),
	}}
	res := ConvertState(st, &KCC{}, Options{GitHubOperator: "git-hubby"})
	if got, want := kindsOf(res), []string{"ActionsOrganizationPermissions", "BranchDefault", "Repository"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kinds %v, want %v", got, want)
	}
}
