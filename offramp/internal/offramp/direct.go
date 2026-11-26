// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"strconv"
	"strings"
)

// Kinds a service mapping does not describe: Config Connector's direct and
// DCL-based kinds (written by hand, with no entry in config/servicemappings,
// and named unlike the Terraform type), and config-connector-extensions'
// kinds for what Config Connector lacks (loaded with --crds into KCC.Ext).
// Each converter here is checked against the Terraform state of a live
// resource (tf-infra's, 2026-11-26).

// tfAlias names the service mapping entry of an upstream Terraform type
// that Config Connector maps under its own fork's name.
var tfAlias = map[string]string{
	"google_project_iam_custom_role":      "google_iam_custom_role",
	"google_organization_iam_custom_role": "google_iam_custom_role",
}

func init() {
	for t, f := range map[string]func(*TFResource, *KCC) ([]*Object, []Finding){
		"google_logging_metric":                        convertLogMetric,
		"google_logging_project_bucket_config":         convertLogBucket("project"),
		"google_logging_folder_bucket_config":          convertLogBucket("folder"),
		"google_logging_organization_bucket_config":    convertLogBucket("organization"),
		"google_logging_billing_account_bucket_config": convertLogBucket("billing_account"),
		"google_cloud_quotas_quota_preference":         convertQuotaPreference,
		"google_cloud_identity_group_membership":       convertGroupMembership,
		"google_cloudbuildv2_repository":               convertBuildRepository,
	} {
		special[t] = f
	}
}

func kindOf(r *TFResource, k *KCC, kind string) (*CRD, []Finding) {
	if c := k.CRDs[kind]; c != nil {
		return c, nil
	}
	return nil, []Finding{{r.Address, r.Type, "gap", "no " + kind + " kind loaded"}}
}

func extKindOf(r *TFResource, k *KCC, kind string) (*CRD, []Finding) {
	if c := k.Ext[kind]; c != nil {
		return c, nil
	}
	return nil, []Finding{{r.Address, r.Type, "gap", "config-connector-extensions' " + kind + " is not loaded: pass its config/crds with --crds"}}
}

// recheckRequired drops generic's "required field has no value" notes for
// the fields a converter filled after generic ran.
func recheckRequired(o *Object, f []Finding) []Finding {
	spec := specOf(o)
	out := f[:0]
	for _, x := range f {
		if x.Class == "note" && strings.Contains(x.Reason, "required field spec.") {
			field := strings.TrimSuffix(x.Reason[strings.Index(x.Reason, "spec.")+5:], " has no value from the state")
			if _, ok := spec[field]; ok {
				continue
			}
		}
		out = append(out, x)
	}
	return out
}

func specOf(o *Object) map[string]any { return o.Obj["spec"].(map[string]any) }
func metaOf(o *Object) map[string]any { return o.Obj["metadata"].(map[string]any) }

// convertLogMetric: google_logging_metric is LoggingLogMetric (direct). Its
// name may hold a slash (a metric in a folder of metrics); the bucket is a
// reference.
func convertLogMetric(r *TFResource, k *KCC) ([]*Object, []Finding) {
	crd, f := kindOf(r, k, "LoggingLogMetric")
	if crd == nil {
		return nil, f
	}
	o, f := generic(r, k, nil, crd)
	if b := str(r.Attrs["bucket_name"]); b != "" {
		specOf(o)["loggingLogBucketRef"] = map[string]any{"external": b}
	}
	return []*Object{o}, recheckRequired(o, f)
}

// convertLogBucket: a log bucket's settings (google_logging_*_bucket_config)
// are LoggingLogBucket (direct), whose resource ID is the bucket's ID and
// whose parent is a project, folder, organization or billing account.
// Terraform's _Default and _Required buckets exist with the parent; Config
// Connector adopts them like any other.
func convertLogBucket(parent string) func(*TFResource, *KCC) ([]*Object, []Finding) {
	return func(r *TFResource, k *KCC) ([]*Object, []Finding) {
		crd, f := kindOf(r, k, "LoggingLogBucket")
		if crd == nil {
			return nil, f
		}
		attrs := map[string]any{}
		for a, v := range r.Attrs {
			if a != "name" && a != parent {
				attrs[a] = v
			}
		}
		rr := *r
		rr.Attrs = attrs
		o, f := generic(&rr, k, nil, crd)
		spec := specOf(o)
		id := str(r.Attrs["bucket_id"])
		spec["resourceID"] = id
		p := str(r.Attrs[parent])
		switch parent {
		case "project":
			spec["projectRef"] = map[string]any{"external": strings.TrimPrefix(p, "projects/")}
		case "folder":
			spec["folderRef"] = map[string]any{"external": strings.TrimPrefix(p, "folders/")}
		case "organization":
			spec["organizationRef"] = map[string]any{"external": strings.TrimPrefix(p, "organizations/")}
		case "billing_account":
			spec["billingAccountRef"] = map[string]any{"external": strings.TrimPrefix(p, "billingAccounts/")}
		}
		metaOf(o)["name"] = sanitize(lastSegment(p) + "-" + id)
		return []*Object{o}, recheckRequired(o, f)
	}
}

// convertQuotaPreference: google_cloud_quotas_quota_preference is
// APIQuotaPreference (direct); the parent is projects/<id>, folders/<n> or
// organizations/<n>. granted_value and the other outputs of quota_config
// have no spec field and are left out.
func convertQuotaPreference(r *TFResource, k *KCC) ([]*Object, []Finding) {
	crd, f := kindOf(r, k, "APIQuotaPreference")
	if crd == nil {
		return nil, f
	}
	o, f := generic(r, k, nil, crd)
	spec := specOf(o)
	parent := str(r.Attrs["parent"])
	kind, id, _ := strings.Cut(parent, "/")
	switch kind {
	case "projects":
		spec["projectRef"] = map[string]any{"external": id}
	case "folders":
		spec["folderRef"] = map[string]any{"external": id}
	case "organizations":
		spec["organizationRef"] = map[string]any{"external": id}
	}
	if str(r.Attrs["ignore_safety_checks"]) != "" && str(r.Attrs["ignore_safety_checks"]) != "QUOTA_SAFETY_CHECK_UNSPECIFIED" {
		f = append(f, Finding{r.Address, r.Type, "note", "ignore_safety_checks is a request option with no spec field: set it again if the preference is raised from Kubernetes"})
	}
	// preferred_value is required; the API leaves a zero out, so the state
	// holds "" for a preference of 0 (no quota), which is still a value.
	if qc, ok := r.Attrs["quota_config"].([]any); ok && len(qc) > 0 {
		m, _ := qc[0].(map[string]any)
		n, _ := strconv.ParseInt(scalarString(m["preferred_value"]), 10, 64)
		cfg := map[string]any{"preferredValue": n}
		if a, ok := m["annotations"].(map[string]any); ok && len(a) > 0 {
			cfg["annotations"] = a
		}
		spec["quotaConfig"] = cfg
	}
	metaOf(o)["name"] = sanitize(id + "-" + str(r.Attrs["name"]))
	return []*Object{o}, recheckRequired(o, f)
}

// convertGroupMembership: google_cloud_identity_group_membership is
// CloudIdentityMembership (DCL); its resource ID is the membership's
// number, its group a reference (groups/<id>).
func convertGroupMembership(r *TFResource, k *KCC) ([]*Object, []Finding) {
	crd, f := kindOf(r, k, "CloudIdentityMembership")
	if crd == nil {
		return nil, f
	}
	o, f := generic(r, k, nil, crd)
	spec := specOf(o)
	spec["resourceID"] = lastSegment(str(r.Attrs["name"]))
	spec["groupRef"] = map[string]any{"external": str(r.Attrs["group"])}
	delete(spec, "projectRef")
	who := ""
	if pk, ok := spec["preferredMemberKey"].(map[string]any); ok {
		who = str(pk["id"])
	}
	metaOf(o)["name"] = sanitize(lastSegment(str(r.Attrs["group"])) + "-" + who)
	return []*Object{o}, recheckRequired(o, f)
}

// convertBuildRepository: google_cloudbuildv2_repository has no Config
// Connector kind; config-connector-extensions' CloudBuildRepository is the
// same resource, its connection the parent connection's name.
func convertBuildRepository(r *TFResource, k *KCC) ([]*Object, []Finding) {
	crd, f := extKindOf(r, k, "CloudBuildRepository")
	if crd == nil {
		return nil, f
	}
	o, f := generic(r, k, nil, crd)
	specOf(o)["connection"] = lastSegment(str(r.Attrs["parent_connection"]))
	delete(specOf(o), "etag") // the server's, of the state's read: stale on the next update
	return []*Object{o}, recheckRequired(o, f)
}

// iamTarget is the resource of an IAM type that no service mapping's
// iamConfig names: its kind, where that kind lives and how the state names
// the resource.
type iamTarget struct {
	kind, apiVersion string
	ext              bool // the IAM kind is config-connector-extensions' (IAMMember/IAMBinding)
	name             func(a map[string]any) string
}

func attr(n string) func(map[string]any) string {
	return func(a map[string]any) string { return str(a[n]) }
}

func projectPath(suffix ...string) func(map[string]any) string {
	return func(a map[string]any) string {
		s := "projects/" + str(a["project"])
		for _, x := range suffix {
			if strings.HasPrefix(x, "@") {
				x = str(a[x[1:]])
			}
			s += x
		}
		return s
	}
}

// iamTargets: Config Connector's external-only kinds (Organization,
// BillingAccount: the bare ID), and IAP's resources, which only
// config-connector-extensions' IAMMember and IAMBinding reference (by the
// IAP name, external).
var iamTargets = map[string]iamTarget{
	"google_organization":    {kind: "Organization", apiVersion: "resourcemanager.cnrm.cloud.google.com/v1beta1", name: attr("org_id")},
	"google_billing_account": {kind: "BillingAccount", apiVersion: "billing.cnrm.cloud.google.com/v1beta1", name: attr("billing_account_id")},

	"google_iap_web":                        {kind: "IAPWeb", ext: true, name: projectPath("/iap_web")},
	"google_iap_web_type_compute":           {kind: "IAPWebTypeCompute", ext: true, name: projectPath("/iap_web/compute")},
	"google_iap_web_backend_service":        {kind: "IAPWebBackendService", ext: true, name: attr("web_backend_service")},
	"google_iap_web_region_backend_service": {kind: "IAPWebRegionBackendService", ext: true, name: attr("web_region_backend_service")},
	"google_iap_web_cloud_run_service":      {kind: "IAPWebCloudRunService", ext: true, name: attr("cloud_run_service_name")},
	"google_iap_web_type_app_engine":        {kind: "IAPWebTypeAppEngine", ext: true, name: projectPath("/iap_web/appengine-", "@app_id")},
	"google_iap_app_engine_service":         {kind: "IAPAppEngineService", ext: true, name: projectPath("/iap_web/appengine-", "@app_id", "/services/", "@service")},
	"google_iap_app_engine_version":         {kind: "IAPAppEngineVersion", ext: true, name: projectPath("/iap_web/appengine-", "@app_id", "/services/", "@service", "/versions/", "@version_id")},
	"google_iap_tunnel":                     {kind: "IAPTunnel", ext: true, name: projectPath("/iap_tunnel")},
	"google_iap_tunnel_instance":            {kind: "IAPTunnelInstance", ext: true, name: projectPath("/iap_tunnel/zones/", "@zone", "/instances/", "@instance")},
	"google_iap_tunnel_dest_group":          {kind: "IAPTunnelDestGroup", ext: true, name: projectPath("/iap_tunnel/locations/", "@region", "/destGroups/", "@dest_group")},
}

// iamTargetOf finds the iamTarget of google_<x>_iam_member or _binding.
func iamTargetOf(tfType string) (iamTarget, bool) {
	base := strings.TrimSuffix(strings.TrimSuffix(tfType, "_iam_member"), "_iam_binding")
	t, ok := iamTargets[base]
	return t, ok
}

// convertIAMTarget converts IAM on an iamTarget. A member is IAMPolicyMember
// (Config Connector's kinds) or IAMMember (IAP: config-connector-extensions);
// a binding is IAMBinding (exact: it owns the role's member list, as
// Terraform's binding does) where IAMBinding takes the kind, else
// IAMPartialPolicy.
func convertIAMTarget(r *TFResource, k *KCC, t iamTarget) (*Object, []Finding) {
	ref := t.name(r.Attrs)
	member := strings.HasSuffix(r.Type, "_iam_member")
	var kind string
	var crd *CRD
	var f []Finding
	resourceRef := map[string]any{"kind": t.kind, "external": ref}
	switch {
	case t.ext && member:
		kind = "IAMMember"
		crd, f = extKindOf(r, k, kind)
	case t.ext || iamBindingKinds[t.kind] && !member:
		// Exact, as Terraform's binding and as convertIAM does.
		kind = "IAMBinding"
		crd, f = extKindOf(r, k, kind)
		if !t.ext {
			resourceRef["apiVersion"] = t.apiVersion
		}
	case member:
		kind = "IAMPolicyMember"
		crd, f = kindOf(r, k, kind)
		resourceRef["apiVersion"] = t.apiVersion
	default:
		kind = "IAMPartialPolicy"
		crd, f = kindOf(r, k, kind)
		resourceRef["apiVersion"] = t.apiVersion
	}
	if crd == nil {
		return nil, f
	}
	role := str(r.Attrs["role"])
	cond := condition(r.Attrs["condition"])
	spec := map[string]any{"resourceRef": resourceRef}
	who := ""
	if member {
		who = str(r.Attrs["member"])
		spec["member"], spec["role"] = who, role
		if cond != nil {
			spec["condition"] = cond
		}
	} else {
		raw, _ := r.Attrs["members"].([]any)
		if kind == "IAMBinding" && len(raw) == 0 {
			return nil, []Finding{{r.Address, r.Type, "refused", "an authoritative binding with no members empties the role; an IAMBinding needs at least one member"}}
		}
		var members []any
		for _, m := range raw {
			who += str(m) + ","
			if kind == "IAMBinding" {
				members = append(members, str(m))
			} else {
				members = append(members, map[string]any{"member": str(m)})
			}
		}
		if kind == "IAMBinding" {
			spec["role"], spec["members"] = role, members
			if cond != nil {
				spec["condition"] = cond
			}
		} else {
			b := map[string]any{"role": role, "members": members}
			if cond != nil {
				b["condition"] = cond
			}
			spec["bindings"] = []any{b}
			f = append(f, Finding{r.Address, r.Type, "note", "an authoritative binding becomes IAMPartialPolicy: the listed members are kept, and members added outside it are no longer removed"})
		}
	}
	obj := map[string]any{
		"apiVersion": crd.APIVersion(),
		"kind":       kind,
		"metadata":   map[string]any{"name": sanitize(lastSegment(ref) + "-" + lastSegment(role) + "-" + shortHash(who+str(cond)))},
		"spec":       spec,
	}
	return &Object{Source: r.Address, Call: r.Call, TFType: r.Type, Kind: kind, Obj: obj, Deps: r.Deps}, f
}
