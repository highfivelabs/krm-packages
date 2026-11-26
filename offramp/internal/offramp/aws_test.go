// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func awsResource(typ, name string, attrs map[string]any) StateResource {
	return StateResource{
		Mode: "managed", Type: typ, Name: name,
		Provider:  `provider["registry.terraform.io/hashicorp/aws"]`,
		Instances: []StateInstance{{Attributes: attrs}},
	}
}

const (
	testInstance = "arn:aws:sso:::instance/ssoins-0000000000000000"
	testPS       = "arn:aws:sso:::permissionSet/ssoins-0000000000000000/ps-1111111111111111"
)

// TestAWSOrganizationsAndSSO converts an organization and Identity Center
// estate in the shapes Terraform's aws provider (v6) keeps them: every
// object adopts (never creates) by its kind's identifier and is retained on
// delete, and the permission set's attachments are inline.
func TestAWSOrganizationsAndSSO(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{
		awsResource("aws_organizations_account", "a", map[string]any{
			"id": "111111111111", "name": "dev", "email": "dev@example.com", "parent_id": "ou-ab12-cdefgh34",
			"tags": map[string]any{"env": "dev"}, "tags_all": map[string]any{"env": "dev", "repo": "infra"}, "close_on_deletion": false,
		}),
		awsResource("aws_organizations_organizational_unit", "ou", map[string]any{
			"id": "ou-ab12-cdefgh34", "name": "dev", "parent_id": "r-ab12",
		}),
		awsResource("aws_organizations_policy", "scp", map[string]any{
			"id": "p-abcdefgh", "name": "deny-leave", "description": "", "type": "SERVICE_CONTROL_POLICY",
			"content": `{"Version":"2012-10-17","Statement":[]}`,
		}),
		awsResource("aws_ssoadmin_permission_set", "ps", map[string]any{
			"arn": testPS, "instance_arn": testInstance, "name": "Admin", "session_duration": "PT12H", "region": "us-east-1",
		}),
		awsResource("aws_ssoadmin_managed_policy_attachment", "mpa", map[string]any{
			"instance_arn": testInstance, "permission_set_arn": testPS, "managed_policy_arn": "arn:aws:iam::aws:policy/AdministratorAccess",
		}),
		awsResource("aws_ssoadmin_permission_set_inline_policy", "inline", map[string]any{
			"instance_arn": testInstance, "permission_set_arn": testPS, "inline_policy": `{"Version":"2012-10-17","Statement":[]}`,
		}),
		awsResource("aws_ssoadmin_account_assignment", "as", map[string]any{
			"instance_arn": testInstance, "permission_set_arn": testPS, "principal_id": "u-1", "principal_type": "USER",
			"target_id": "111111111111", "target_type": "AWS_ACCOUNT", "region": "us-east-1",
		}),
		awsResource("aws_ssoadmin_managed_policy_attachment", "elsewhere", map[string]any{
			"instance_arn": testInstance, "permission_set_arn": "arn:aws:sso:::permissionSet/ssoins-0000000000000000/ps-other", "managed_policy_arn": "arn:aws:iam::aws:policy/ReadOnlyAccess",
		}),
	}}
	res := ConvertState(st, &KCC{}, Options{})

	kinds := map[string]int{}
	byKind := map[string]map[string]any{}
	for _, o := range res.Objects {
		kinds[o.Kind]++
		byKind[o.Kind] = o.Obj
		ann := dig(o.Obj, "metadata", "annotations").(map[string]any)
		if ann[AnnACKAdoptionPolicy] != "adopt" || ann[AnnACKDeletionPolicy] != "retain" {
			t.Errorf("%s: annotations %v", o.Kind, ann)
		}
		if ann[AnnDeletionPolicy] != nil || ann[AnnStateIntoSpec] != nil {
			t.Errorf("%s: Config Connector annotations on an ACK object", o.Kind)
		}
		var fields map[string]string
		if err := json.Unmarshal([]byte(str(ann[AnnACKAdoptionFields])), &fields); err != nil || len(fields) == 0 {
			t.Errorf("%s: adoption-fields %v", o.Kind, ann[AnnACKAdoptionFields])
		}
	}
	want := map[string]int{"Account": 1, "AccountPlacement": 1, "OrganizationalUnit": 1, "Policy": 1, "PermissionSet": 1, "Assignment": 1}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("kinds %v, want %v", kinds, want)
	}

	var gaps, notes []Finding
	for _, f := range res.Findings {
		switch f.Class {
		case "gap":
			gaps = append(gaps, f)
		case "note":
			notes = append(notes, f)
		default:
			t.Errorf("unexpected finding %+v", f)
		}
	}
	if len(gaps) != 1 || !strings.Contains(gaps[0].Reason, "ps-other") {
		t.Errorf("want one gap for the attachment outside the state, got %+v", gaps)
	}
	if len(notes) != 1 || notes[0].Type != "aws_organizations_policy" {
		t.Errorf("want one note on the policy's missing description, got %+v", notes)
	}

	for _, c := range []struct {
		kind string
		path []string
		want any
	}{
		{"Account", []string{"metadata", "annotations", AnnACKAdoptionFields}, `{"accountID":"111111111111"}`},
		{"Account", []string{"spec", "tags"}, []any{map[string]any{"key": "env", "value": "dev"}, map[string]any{"key": "repo", "value": "infra"}}},
		{"AccountPlacement", []string{"spec", "parentID"}, "ou-ab12-cdefgh34"},
		{"OrganizationalUnit", []string{"metadata", "annotations", AnnACKAdoptionFields}, `{"id":"ou-ab12-cdefgh34"}`},
		{"Policy", []string{"spec", "type_"}, "SERVICE_CONTROL_POLICY"},
		{"PermissionSet", []string{"spec", "managedPolicies"}, []any{"arn:aws:iam::aws:policy/AdministratorAccess"}},
		{"PermissionSet", []string{"spec", "inlinePolicy", "Version"}, "2012-10-17"},
		{"PermissionSet", []string{"metadata", "annotations", AnnACKRegion}, "us-east-1"},
		{"PermissionSet", []string{"metadata", "annotations", AnnACKAdoptionFields}, `{"identifier":"` + testInstance + `|` + testPS + `"}`},
		{"Assignment", []string{"metadata", "annotations", AnnACKAdoptionFields}, `{"identifier":"` + testInstance + `|111111111111|AWS_ACCOUNT|` + testPS + `|USER|u-1"}`},
	} {
		if got := dig(byKind[c.kind], c.path...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %s = %#v, want %#v", c.kind, strings.Join(c.path, "."), got, c.want)
		}
	}
}

// TestAWSUnmappedIsAGap: an aws type the converter has no mapping for is a
// gap that says where to add it, not a provider-wide refusal.
func TestAWSUnmappedIsAGap(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{awsResource("aws_vpc", "v", map[string]any{"id": "vpc-1"})}}
	res := ConvertState(st, &KCC{}, Options{})
	if len(res.Objects) != 0 || len(res.Findings) != 1 || res.Findings[0].Class != "gap" || !strings.Contains(res.Findings[0].Reason, "aws.go") {
		t.Errorf("objects %d, findings %+v", len(res.Objects), res.Findings)
	}
}

// TestProvidersFollowTheObjects: the Environment pins the providers the
// objects need (Bundt's names), Config Connector only when there is one of
// its kinds, and a provider no --provider pins is a note, not a guess.
func TestProvidersFollowTheObjects(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{
		awsResource("aws_organizations_organizational_unit", "ou", map[string]any{"id": "ou-1", "name": "dev", "parent_id": "r-1"}),
		awsResource("aws_organizations_policy", "scp", map[string]any{"id": "p-1", "name": "p", "description": "d", "type": "SERVICE_CONTROL_POLICY", "content": "{}"}),
	}}
	res := ConvertState(st, &KCC{}, Options{})
	b := Group(res.Objects, BundtOptions{Name: "org", Team: "platform", Env: "global", Target: "t", Version: "0.1.0", Namespace: "n",
		KCCVersion: "1.160.0", Providers: map[string]string{"aws.ack-organizations": "1.7.0"}})
	var env map[string]any
	for _, d := range b.Record {
		if d["kind"] == "Environment" {
			env = d
		}
	}
	if got, want := dig(env, "spec", "providers"), map[string]any{"aws.ack-organizations": "1.7.0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("providers %v, want %v", got, want)
	}
	found := false
	for _, n := range b.Notes {
		if strings.Contains(n, "hfl.ack-extensions") {
			found = true
		}
	}
	if !found {
		t.Errorf("no note for the unpinned hfl.ack-extensions: %v", b.Notes)
	}
}
