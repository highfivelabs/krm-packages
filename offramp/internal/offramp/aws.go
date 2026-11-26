// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"encoding/json"
	"sort"
	"strings"
)

// AWS: Terraform's aws provider to AWS Controllers for Kubernetes (ACK)
// objects, and ack-extensions' where ACK has no kind (docs/aws.md, the
// module map). An object adopts the existing resource with ACK's own
// annotations (ack-extensions docs/slot-moves.md, "The annotations"):
// adoption-policy adopt finds the resource by the kind's identifier
// (adoption-fields) and fills the spec from AWS, so nothing is created
// and the adoption makes no write; deletion-policy retain keeps the
// resource when the object is deleted. The spec is still written from the
// state, so a render compares field for field.

// ACK annotations.
const (
	AnnACKAdoptionPolicy = "services.k8s.aws/adoption-policy"
	AnnACKAdoptionFields = "services.k8s.aws/adoption-fields"
	AnnACKDeletionPolicy = "services.k8s.aws/deletion-policy"
	AnnACKRegion         = "services.k8s.aws/region"
)

// isACK says whether an apiVersion is ACK's or ack-extensions'.
func isACK(apiVersion string) bool {
	g, _, _ := strings.Cut(apiVersion, "/")
	return strings.HasSuffix(g, ".services.k8s.aws") || strings.HasSuffix(g, ".aws.hfl.dev")
}

// awsContext is what one AWS converter needs from the rest of the state:
// the resources Terraform keeps apart and the kind holds inline.
type awsContext struct {
	// managedPolicies, inlinePolicy and customerPolicies are a permission
	// set's attachments, by permission set ARN.
	managedPolicies  map[string][]string
	inlinePolicy     map[string]string
	customerPolicies map[string][]any
	// permissionSets are the permission set ARNs converted from this state.
	permissionSets map[string]bool
}

func newAWSContext(rs []*TFResource) *awsContext {
	c := &awsContext{managedPolicies: map[string][]string{}, inlinePolicy: map[string]string{}, customerPolicies: map[string][]any{}, permissionSets: map[string]bool{}}
	for _, r := range rs {
		ps := str(r.Attrs["permission_set_arn"])
		switch r.Type {
		case "aws_ssoadmin_permission_set":
			c.permissionSets[str(r.Attrs["arn"])] = true
		case "aws_ssoadmin_managed_policy_attachment":
			c.managedPolicies[ps] = append(c.managedPolicies[ps], str(r.Attrs["managed_policy_arn"]))
		case "aws_ssoadmin_permission_set_inline_policy":
			c.inlinePolicy[ps] = str(r.Attrs["inline_policy"])
		case "aws_ssoadmin_customer_managed_policy_attachment":
			for _, ref := range list(r.Attrs["customer_managed_policy_reference"]) {
				m, _ := ref.(map[string]any)
				out := map[string]any{"name": str(m["name"])}
				if p := str(m["path"]); p != "" {
					out["path"] = p
				}
				c.customerPolicies[ps] = append(c.customerPolicies[ps], out)
			}
		}
	}
	for _, m := range []map[string][]string{c.managedPolicies} {
		for k := range m {
			sort.Strings(m[k])
		}
	}
	return c
}

type awsConverter func(r *TFResource, c *awsContext) ([]*Object, []Finding)

var awsConverters = map[string]awsConverter{
	"aws_organizations_account":             convertAWSAccount,
	"aws_organizations_organizational_unit": convertAWSOU,
	"aws_organizations_policy":              convertAWSOrgPolicy,
	"aws_organizations_policy_attachment":   convertAWSPolicyAttachment,
	"aws_ssoadmin_permission_set":           convertAWSPermissionSet,
	"aws_ssoadmin_account_assignment":       convertAWSAssignment,
	// Folded into their PermissionSet (newAWSContext).
	"aws_ssoadmin_managed_policy_attachment":          foldIntoPermissionSet("managedPolicies"),
	"aws_ssoadmin_permission_set_inline_policy":       foldIntoPermissionSet("inlinePolicy"),
	"aws_ssoadmin_customer_managed_policy_attachment": foldIntoPermissionSet("customerManagedPolicyReferences"),
}

// convertAWS converts one aws resource, or says why it does not.
func convertAWS(r *TFResource, c *awsContext) ([]*Object, []Finding) {
	f, ok := awsConverters[r.Type]
	if !ok {
		return nil, []Finding{{r.Address, r.Type, "gap", "no AWS mapping in the converter yet (docs/aws.md names the ACK or ack-extensions kind): keep it in Terraform, or add it to aws.go"}}
	}
	return f(r, c)
}

// ackObject is one ACK object that adopts the resource fields names.
func ackObject(r *TFResource, apiVersion, kind, name string, spec map[string]any, fields map[string]string) *Object {
	b, _ := json.Marshal(fields)
	ann := map[string]any{
		AnnACKAdoptionPolicy: "adopt",
		AnnACKAdoptionFields: string(b),
	}
	if region := str(r.Attrs["region"]); region != "" {
		ann[AnnACKRegion] = region
	}
	for k, v := range spec {
		if empty(v) {
			delete(spec, k)
		}
	}
	return &Object{
		Source: r.Address, Call: r.Call, TFType: r.Type, Kind: kind, Deps: r.Deps,
		Obj: map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata":   map[string]any{"name": sanitize(name), "annotations": ann},
			"spec":       spec,
		},
	}
}

// ackTags is Terraform's tags map as ACK's [{key, value}], sorted by key.
// tags_all (the provider's default tags merged in) is what AWS holds, so it
// wins when present.
func ackTags(attrs map[string]any) []any {
	m, _ := attrs["tags_all"].(map[string]any)
	if len(m) == 0 {
		m, _ = attrs["tags"].(map[string]any)
	}
	var out []any
	for _, k := range sortedKeys(m) {
		out = append(out, map[string]any{"key": k, "value": str(m[k])})
	}
	return out
}

func list(v any) []any {
	l, _ := v.([]any)
	return l
}

const (
	ackOrganizations = "organizations.services.k8s.aws/v1alpha1"
	extOrganizations = "organizations.aws.hfl.dev/v1alpha1"
	extSSO           = "sso.cloudcontrol.aws.hfl.dev/v1alpha1"
)

// convertAWSAccount: a member account is ACK's Account (organizations-
// controller), adopted by its account ID; the OU it sits in is
// ack-extensions' AccountPlacement, since ACK's Account has no parent.
func convertAWSAccount(r *TFResource, _ *awsContext) ([]*Object, []Finding) {
	a := r.Attrs
	id := str(a["id"])
	spec := map[string]any{
		"name":                   str(a["name"]),
		"email":                  str(a["email"]),
		"roleName":               str(a["role_name"]),
		"iamUserAccessToBilling": str(a["iam_user_access_to_billing"]),
		"tags":                   ackTags(a),
	}
	acct := ackObject(r, ackOrganizations, "Account", str(a["name"]), spec, map[string]string{"accountID": id})
	objs := []*Object{acct}
	var f []Finding
	if p := str(a["parent_id"]); p != "" {
		pl := ackObject(r, extOrganizations, "AccountPlacement", str(a["name"])+"-placement",
			map[string]any{"accountID": id, "parentID": p}, map[string]string{"accountID": id})
		objs = append(objs, pl)
	}
	if b, ok := a["close_on_deletion"].(bool); ok && b {
		f = append(f, Finding{r.Address, r.Type, "note", "close_on_deletion has no ACK field: deleting the Account object with deletion-policy retain leaves the account open"})
	}
	return objs, f
}

// convertAWSOU: ACK's OrganizationalUnit, adopted by its ID.
func convertAWSOU(r *TFResource, _ *awsContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{"name": str(a["name"]), "parentID": str(a["parent_id"]), "tags": ackTags(a)}
	return []*Object{ackObject(r, ackOrganizations, "OrganizationalUnit", "ou-"+str(a["name"]), spec, map[string]string{"id": str(a["id"])})}, nil
}

// convertAWSOrgPolicy: ack-extensions' Policy (ACK has none), adopted by
// its ID. Its description is required; Terraform's is optional.
func convertAWSOrgPolicy(r *TFResource, _ *awsContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{
		"name":        str(a["name"]),
		"description": str(a["description"]),
		"content":     str(a["content"]),
		"type_":       str(a["type"]),
		"tags":        ackTags(a),
	}
	var f []Finding
	if spec["description"] == "" {
		f = append(f, Finding{r.Address, r.Type, "note", "Policy's description is required and the policy has none: adoption fills it from AWS (empty); give it one in the object before changing anything else"})
	}
	return []*Object{ackObject(r, extOrganizations, "Policy", str(a["name"]), spec, map[string]string{"id": str(a["id"])})}, f
}

// convertAWSPolicyAttachment: ack-extensions' PolicyAttachment, one policy
// on one root, OU or account; its ID is policyID and targetID.
func convertAWSPolicyAttachment(r *TFResource, _ *awsContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{"policyID": str(a["policy_id"]), "targetID": str(a["target_id"])}
	return []*Object{ackObject(r, extOrganizations, "PolicyAttachment", str(a["policy_id"])+"-"+str(a["target_id"]), spec,
		map[string]string{"policyID": str(a["policy_id"]), "targetID": str(a["target_id"])})}, nil
}

// convertAWSPermissionSet: AWS::SSO::PermissionSet through ack-extensions'
// Cloud Control manager, adopted by its primary identifier (InstanceArn,
// PermissionSetArn). Terraform's attachments are inline here.
func convertAWSPermissionSet(r *TFResource, c *awsContext) ([]*Object, []Finding) {
	a := r.Attrs
	arn := str(a["arn"])
	spec := map[string]any{
		"instanceArn":     str(a["instance_arn"]),
		"name":            str(a["name"]),
		"description":     str(a["description"]),
		"sessionDuration": str(a["session_duration"]),
		"relayStateType":  str(a["relay_state"]),
		"tags":            ackTags(a),
	}
	if p := c.managedPolicies[arn]; len(p) > 0 {
		var l []any
		for _, x := range p {
			l = append(l, x)
		}
		spec["managedPolicies"] = l
	}
	if p := c.inlinePolicy[arn]; p != "" {
		var doc any
		if json.Unmarshal([]byte(p), &doc) == nil {
			spec["inlinePolicy"] = doc
		}
	}
	if p := c.customerPolicies[arn]; len(p) > 0 {
		spec["customerManagedPolicyReferences"] = p
	}
	return []*Object{ackObject(r, extSSO, "PermissionSet", str(a["name"]), spec,
		map[string]string{"identifier": str(a["instance_arn"]) + "|" + arn})}, nil
}

// foldIntoPermissionSet: an attachment Terraform keeps apart is a field of
// the PermissionSet; nothing to write when that permission set is in the
// state, a gap when it is not.
func foldIntoPermissionSet(field string) awsConverter {
	return func(r *TFResource, c *awsContext) ([]*Object, []Finding) {
		ps := str(r.Attrs["permission_set_arn"])
		if c.permissionSets[ps] {
			return nil, nil
		}
		return nil, []Finding{{r.Address, r.Type, "gap", "attached to a permission set outside this state (" + ps + "): add it to that PermissionSet's " + field}}
	}
}

// convertAWSAssignment: AWS::SSO::Assignment, adopted by its primary
// identifier, every field of which is the spec.
func convertAWSAssignment(r *TFResource, _ *awsContext) ([]*Object, []Finding) {
	a := r.Attrs
	spec := map[string]any{
		"instanceArn":      str(a["instance_arn"]),
		"targetId":         str(a["target_id"]),
		"targetType":       str(a["target_type"]),
		"permissionSetArn": str(a["permission_set_arn"]),
		"principalType":    str(a["principal_type"]),
		"principalId":      str(a["principal_id"]),
	}
	id := strings.Join([]string{spec["instanceArn"].(string), spec["targetId"].(string), spec["targetType"].(string),
		spec["permissionSetArn"].(string), spec["principalType"].(string), spec["principalId"].(string)}, "|")
	name := lastSegment(spec["permissionSetArn"].(string)) + "-" + spec["targetId"].(string) + "-" + shortHash(spec["principalId"].(string))
	return []*Object{ackObject(r, extSSO, "Assignment", name, spec, map[string]string{"identifier": id})}, nil
}
