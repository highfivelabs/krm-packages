// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"encoding/base64"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func ghResource(typ, name string, attrs map[string]any) StateResource {
	return StateResource{
		Mode: "managed", Type: typ, Name: name,
		Provider:  `provider["registry.terraform.io/integrations/github"]`,
		Instances: []StateInstance{{Attributes: attrs}},
	}
}

// TestGitHubRepositoryConfiguration converts a repository's configuration
// in the shapes Terraform's integrations/github provider keeps it: every
// object is abandoned on delete, a CRUD kind adopts by the ID status.id
// would hold, and the owner comes from the repository's full name.
func TestGitHubRepositoryConfiguration(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{
		ghResource("github_repository", "repo", map[string]any{"name": "app", "full_name": "acme/app"}),
		ghResource("github_branch_default", "d", map[string]any{"repository": "app", "branch": "main"}),
		ghResource("github_branch_protection", "p", map[string]any{
			"id": "BPR_1", "repository_id": "R_1", "pattern": "main", "enforce_admins": true, "allows_deletions": false,
			"required_pull_request_reviews": []any{map[string]any{"required_approving_review_count": float64(1), "dismiss_stale_reviews": true}},
			"required_status_checks":        []any{map[string]any{"strict": true, "contexts": []any{"ci"}}},
			"restrict_pushes":               []any{map[string]any{"push_allowances": []any{"/octocat"}}},
		}),
		ghResource("github_actions_variable", "v", map[string]any{"repository": "app", "variable_name": "REGION", "value": "us"}),
		ghResource("github_repository_file", "f", map[string]any{"repository": "app", "file": "README.md", "branch": "main", "content": "hello\n", "commit_message": "m"}),
		ghResource("github_workflow_repository_permissions", "w", map[string]any{"repository": "app", "default_workflow_permissions": "read", "can_approve_pull_request_reviews": false}),
	}}
	res := ConvertState(st, &KCC{}, Options{})
	got := map[string]map[string]any{}
	for _, o := range res.Objects {
		got[o.Kind] = o.Obj
		if isGitHubby(str(o.Obj["apiVersion"])) {
			for _, a := range []string{AnnRESTDeletionPolicy, AnnDeletionPolicy, AnnStateIntoSpec} {
				if dig(o.Obj, "metadata", "annotations", a) != nil {
					t.Errorf("%s: %s on a git-hubby object, which has no per-object deletion policy", o.Kind, a)
				}
			}
			continue
		}
		if dig(o.Obj, "metadata", "annotations", AnnRESTDeletionPolicy) != "abandon" {
			t.Errorf("%s: not abandoned on delete", o.Kind)
		}
		if dig(o.Obj, "metadata", "annotations", AnnDeletionPolicy) != nil {
			t.Errorf("%s: Config Connector annotations on a GitHub object", o.Kind)
		}
	}
	var gaps, notes []string
	for _, f := range res.Findings {
		switch f.Class {
		case "gap":
			gaps = append(gaps, f.Type)
		case "note":
			notes = append(notes, f.Reason)
		}
	}
	if len(gaps) != 0 {
		t.Errorf("gaps %v, want none", gaps)
	}
	want := []string{"/octocat"}
	if len(notes) != len(want) {
		t.Errorf("notes %v, want %d", notes, len(want))
	}
	for i, w := range want {
		if i < len(notes) && !strings.Contains(notes[i], w) {
			t.Errorf("note %d = %q, want it to say %q", i, notes[i], w)
		}
	}
	if _, ok := got["BranchDefault"]; !ok {
		t.Error("no BranchDefault: the default branch is its own kind, as github_branch_default is its own resource")
	}
	for _, c := range []struct {
		kind string
		path []string
		want any
	}{
		{"Repository", []string{"apiVersion"}, "repository.github.hfl.dev/v1alpha1"},
		{"Repository", []string{"spec", "owner"}, "acme"},
		{"Repository", []string{"spec", "name"}, "app"},
		{"Repository", []string{"metadata", "annotations", AnnRESTAdoptID}, "app"},
		{"BranchDefault", []string{"spec", "default_branch"}, "main"},
		{"ActionsVariable", []string{"metadata", "annotations", AnnRESTAdoptID}, "REGION"},
		{"BranchProtection", []string{"metadata", "annotations", AnnRESTAdoptID}, "BPR_1"},
		{"BranchProtection", []string{"spec", "isAdminEnforced"}, true},
		{"BranchProtection", []string{"spec", "allowsDeletions"}, false},
		{"BranchProtection", []string{"spec", "requiresApprovingReviews"}, true},
		{"BranchProtection", []string{"spec", "requiredApprovingReviewCount"}, int64(1)},
		{"BranchProtection", []string{"spec", "requiredStatusCheckContexts"}, []any{"ci"}},
		{"BranchProtection", []string{"spec", "restrictsPushes"}, true},
		{"RepositoryFile", []string{"spec", "content"}, base64.StdEncoding.EncodeToString([]byte("hello\n"))},
		{"WorkflowRepositoryPermissions", []string{"spec", "can_approve_pull_request_reviews"}, false},
	} {
		if v := dig(got[c.kind], c.path...); !reflect.DeepEqual(v, c.want) {
			t.Errorf("%s %s = %#v, want %#v", c.kind, strings.Join(c.path, "."), v, c.want)
		}
	}
}

// TestGitHubNeedsAnOwner: with no repository in the state the owner is
// unknown, and a repository-level resource is a gap that says so.
func TestGitHubNeedsAnOwner(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{
		ghResource("github_actions_variable", "v", map[string]any{"repository": "app", "variable_name": "X", "value": "1"}),
	}}
	res := ConvertState(st, &KCC{}, Options{})
	if len(res.Objects) != 0 || len(res.Findings) != 1 || !strings.Contains(res.Findings[0].Reason, "owner") {
		t.Errorf("objects %d, findings %+v", len(res.Objects), res.Findings)
	}
}

// TestGitHubObjectsMatchTheCRDs: by default every GitHub object is
// github-krm-extensions', and each field it writes is in that kind's CRD
// (crds/github-krm-extensions, copied from the repository), with every
// required one there.
func TestGitHubObjectsMatchTheCRDs(t *testing.T) {
	crds := map[string]*CRD{}
	files, _ := filepath.Glob("../../../crds/github-krm-extensions/*.yaml")
	if len(files) == 0 {
		t.Fatal("no CRDs in crds/github-krm-extensions")
	}
	for _, f := range files {
		c, err := loadCRD(f)
		if err != nil || c == nil {
			t.Fatalf("%s: %v", f, err)
		}
		crds[c.APIVersion()+"/"+c.Kind] = c
	}
	res := ConvertState(githubbyState(), &KCC{}, Options{})
	if g := findingsOf(res, "gap"); len(g) != 0 {
		t.Errorf("gaps: %v", g)
	}
	want := []string{"ActionsOrganizationPermissions", "ActionsOrganizationWorkflowPermissions", "BranchDefault", "OrganizationSettings", "Repository", "Repository", "RepositoryTopics", "RepositoryVulnerabilityAlerts"}
	if got := kindsOf(res); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds %v, want %v", got, want)
	}
	for _, o := range res.Objects {
		av := str(o.Obj["apiVersion"])
		c, ok := crds[av+"/"+o.Kind]
		if !ok {
			t.Errorf("%s %s: no CRD", av, o.Kind)
			continue
		}
		for _, e := range schemaErrors(o.Obj["spec"], c.Spec, "spec") {
			t.Errorf("%s %s: %s", o.Kind, o.Name(), e)
		}
	}
	site := objectNamed(res, "Repository", "site")
	for _, c := range []struct {
		path []string
		want any
	}{
		{[]string{"spec", "homepage"}, "https://acme.example"},
		{[]string{"spec", "has_issues"}, false},
		{[]string{"spec", "allow_rebase_merge"}, false},
		{[]string{"metadata", "annotations", AnnRESTAdoptID}, "site"},
	} {
		if v := dig(site, c.path...); !reflect.DeepEqual(v, c.want) {
			t.Errorf("site %s = %#v, want %#v", strings.Join(c.path, "."), v, c.want)
		}
	}
	if v := dig(objectNamed(res, "OrganizationSettings", "acme"), "spec", "billing_email"); v != "billing@acme.example" {
		t.Errorf("OrganizationSettings billing_email = %#v", v)
	}
}
