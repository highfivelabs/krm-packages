// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"reflect"
	"strings"
	"testing"
)

// managed is one managed resource of the google provider, one instance.
func managed(typ, name string, attrs map[string]any) StateResource {
	return StateResource{
		Mode: "managed", Type: typ, Name: name,
		Provider:  `provider["registry.terraform.io/hashicorp/google"]`,
		Instances: []StateInstance{{Attributes: attrs}},
	}
}

// TestDirectKinds converts, in the shape a Terraform state holds them, the
// types Config Connector maps with no service mapping (direct and DCL kinds)
// and those only config-connector-extensions has. Each object is what Config
// Connector (or the extensions) would adopt with no write.
func TestDirectKinds(t *testing.T) {
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	if err := k.LoadExtensions(vendoredExt); err != nil {
		t.Fatal(err)
	}
	st := &State{Version: 4, Resources: []StateResource{
		managed("google_project_iam_custom_role", "role", map[string]any{
			"name": "projects/p/roles/selfDelete", "role_id": "selfDelete", "project": "p",
			"title": "Self delete", "permissions": []any{"compute.instances.delete"}, "stage": "GA", "deleted": false,
		}),
		managed("google_logging_metric", "m", map[string]any{
			"name": "proxy/failovers", "project": "p", "filter": `jsonPayload.event="failover"`,
			"label_extractors":  map[string]any{"reason": "EXTRACT(jsonPayload.reason)"},
			"metric_descriptor": []any{map[string]any{"metric_kind": "DELTA", "value_type": "INT64", "unit": "1"}},
		}),
		managed("google_logging_project_bucket_config", "b", map[string]any{
			"name": "projects/p/locations/global/buckets/_Default", "project": "p", "bucket_id": "_Default",
			"location": "global", "retention_days": float64(30), "enable_analytics": true,
		}),
		managed("google_cloud_quotas_quota_preference", "q", map[string]any{
			"name": "gpu-zero", "parent": "projects/p", "service": "run.googleapis.com", "quota_id": "GpuPerRegion",
			"dimensions":           map[string]any{"region": "us-central1"},
			"quota_config":         []any{map[string]any{"preferred_value": "", "granted_value": "0"}},
			"ignore_safety_checks": "QUOTA_SAFETY_CHECK_UNSPECIFIED",
		}),
		managed("google_cloud_identity_group_membership", "gm", map[string]any{
			"name": "groups/g1/memberships/123", "group": "groups/g1",
			"preferred_member_key": []any{map[string]any{"id": "sa@p.iam.gserviceaccount.com", "namespace": ""}},
			"roles":                []any{map[string]any{"name": "MEMBER", "expiry_detail": []any{}}},
		}),
		managed("google_cloudbuildv2_repository", "repo", map[string]any{
			"name": "app", "project": "p", "location": "us-central1", "parent_connection": "conn",
			"remote_uri": "https://github.com/o/app.git", "etag": "abc",
		}),
		managed("google_organization_iam_member", "org", map[string]any{
			"org_id": "123", "role": "roles/billing.creator", "member": "group:g@example.com",
		}),
		managed("google_billing_account_iam_member", "bill", map[string]any{
			"billing_account_id": "000000-111111-222222", "role": "roles/billing.admin", "member": "group:g@example.com",
		}),
		managed("google_iap_web_backend_service_iam_member", "iap", map[string]any{
			"project": "p", "web_backend_service": "projects/p/iap_web/compute/services/web",
			"role": "roles/iap.httpsResourceAccessor", "member": "group:g@example.com",
		}),
		managed("google_iap_web_cloud_run_service_iam_binding", "iaprun", map[string]any{
			"project": "p", "location": "us-central1", "cloud_run_service_name": "projects/p/iap_web/cloud_run-us-central1/services/app",
			"role": "roles/iap.httpsResourceAccessor", "members": []any{"group:g@example.com"},
		}),
	}}
	res := ConvertState(st, k, Options{})
	for _, f := range res.Findings {
		t.Errorf("unexpected finding %+v", f)
	}
	got := map[string]map[string]any{}
	for _, o := range res.Objects {
		got[o.TFType] = o.Obj
	}
	for _, c := range []struct {
		tfType, kind string
		path         []string
		want         any
	}{
		{"google_project_iam_custom_role", "IAMCustomRole", []string{"spec", "resourceID"}, "selfDelete"},
		{"google_project_iam_custom_role", "IAMCustomRole", []string{"metadata", "annotations", "cnrm.cloud.google.com/project-id"}, "p"},
		{"google_logging_metric", "LoggingLogMetric", []string{"spec", "resourceID"}, "proxy/failovers"},
		{"google_logging_metric", "LoggingLogMetric", []string{"spec", "metricDescriptor", "valueType"}, "INT64"},
		{"google_logging_metric", "LoggingLogMetric", []string{"spec", "labelExtractors", "reason"}, "EXTRACT(jsonPayload.reason)"},
		{"google_logging_project_bucket_config", "LoggingLogBucket", []string{"spec", "resourceID"}, "_Default"},
		{"google_logging_project_bucket_config", "LoggingLogBucket", []string{"spec", "projectRef", "external"}, "p"},
		{"google_logging_project_bucket_config", "LoggingLogBucket", []string{"spec", "retentionDays"}, int64(30)},
		{"google_cloud_quotas_quota_preference", "APIQuotaPreference", []string{"spec", "quotaConfig", "preferredValue"}, int64(0)},
		{"google_cloud_quotas_quota_preference", "APIQuotaPreference", []string{"spec", "projectRef", "external"}, "p"},
		{"google_cloud_quotas_quota_preference", "APIQuotaPreference", []string{"spec", "quotaID"}, "GpuPerRegion"},
		{"google_cloud_identity_group_membership", "CloudIdentityMembership", []string{"spec", "resourceID"}, "123"},
		{"google_cloud_identity_group_membership", "CloudIdentityMembership", []string{"spec", "groupRef", "external"}, "groups/g1"},
		{"google_cloudbuildv2_repository", "CloudBuildRepository", []string{"spec", "connection"}, "conn"},
		{"google_cloudbuildv2_repository", "CloudBuildRepository", []string{"spec", "etag"}, nil},
		{"google_organization_iam_member", "IAMPolicyMember", []string{"spec", "resourceRef", "kind"}, "Organization"},
		{"google_organization_iam_member", "IAMPolicyMember", []string{"spec", "resourceRef", "external"}, "123"},
		{"google_billing_account_iam_member", "IAMPolicyMember", []string{"spec", "resourceRef", "kind"}, "BillingAccount"},
		{"google_iap_web_backend_service_iam_member", "IAMMember", []string{"spec", "resourceRef", "kind"}, "IAPWebBackendService"},
		{"google_iap_web_backend_service_iam_member", "IAMMember", []string{"spec", "resourceRef", "external"}, "projects/p/iap_web/compute/services/web"},
		{"google_iap_web_cloud_run_service_iam_binding", "IAMBinding", []string{"spec", "members"}, []any{"group:g@example.com"}},
	} {
		o := got[c.tfType]
		if o == nil {
			t.Errorf("%s: not converted", c.tfType)
			continue
		}
		if o["kind"] != c.kind {
			t.Errorf("%s: kind %v, want %s", c.tfType, o["kind"], c.kind)
		}
		if v := dig(o, c.path...); !reflect.DeepEqual(v, c.want) {
			t.Errorf("%s: %s = %#v, want %#v", c.tfType, strings.Join(c.path, "."), v, c.want)
		}
	}
}

// TestExtensionKindsNeedCRDs: without --crds, the types only
// config-connector-extensions has are gaps that say how to load them.
func TestExtensionKindsNeedCRDs(t *testing.T) {
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	st := &State{Version: 4, Resources: []StateResource{
		managed("google_iap_web_backend_service_iam_member", "iap", map[string]any{
			"project": "p", "web_backend_service": "projects/p/iap_web/compute/services/web",
			"role": "roles/iap.httpsResourceAccessor", "member": "group:g@example.com",
		}),
	}}
	res := ConvertState(st, k, Options{})
	if len(res.Objects) != 0 || len(res.Findings) != 1 || res.Findings[0].Class != "gap" || !strings.Contains(res.Findings[0].Reason, "--crds") {
		t.Errorf("objects %d, findings %+v", len(res.Objects), res.Findings)
	}
}

// TestOrganizationBindingIsExact: an organization's binding is IAMBinding,
// as convertIAM makes a project's; a billing account's, which IAMBinding
// does not take, is IAMPartialPolicy.
func TestOrganizationBindingIsExact(t *testing.T) {
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	if err := k.LoadExtensions(vendoredExt); err != nil {
		t.Fatal(err)
	}
	st := &State{Version: 4, Resources: []StateResource{
		managed("google_organization_iam_binding", "org", map[string]any{
			"org_id": "123", "role": "roles/viewer", "members": []any{"group:g@example.com"},
		}),
		managed("google_billing_account_iam_binding", "bill", map[string]any{
			"billing_account_id": "000000-111111-222222", "role": "roles/billing.viewer", "members": []any{"group:g@example.com"},
		}),
	}}
	res := ConvertState(st, k, Options{})
	kinds := map[string]string{}
	for _, o := range res.Objects {
		kinds[o.TFType] = o.Kind
	}
	if kinds["google_organization_iam_binding"] != "IAMBinding" || kinds["google_billing_account_iam_binding"] != "IAMPartialPolicy" {
		t.Errorf("kinds %v", kinds)
	}
}

// vendoredExt is the catalog's copy of config-connector-extensions' CRDs.
const vendoredExt = "../../../crds/config-connector-extensions"
