// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"strings"
)

// Azure: Terraform's azapi provider (what the Azure Verified Modules use)
// to Azure Service Operator (ASO) objects, and aso-extensions' where ASO
// has no kind, in the API versions the catalog's Azure charts render
// (docs/azure.md, the module map).
//
// An azapi resource names its ARM type and API version and sends a body,
// and ASO's spec is that body with `properties` lifted to the top: the
// converter lifts it, turns ARM IDs into ASO references ({id} and
// ...Id fields), and names the parent as the owner (by the object's name
// when the parent is converted too, by ARM ID when it is not).
//
// Adoption: ASO adopts an existing resource by its name and owner; there
// is no adopt annotation. Its first reconcile writes the spec to Azure (a
// PUT of what is already there), so the reconcile policy decides what the
// adoption does (Options.AzureReconcilePolicy):
//
//	skip              ASO only reads: no write, no delete. The safe first
//	                  step: check every object goes Ready with the status
//	                  Azure reports, then move on.
//	detach-on-delete  ASO manages the resource and leaves it on delete
//	                  (the default with DeletionPolicy abandon, as the
//	                  Azure charts set it).
//	manage            ASO manages and deletes (DeletionPolicy delete).
//
// ASO takes the subscription from its credential, not from the object, so
// every object carries hfl.dev/azure-subscription and a state that spans
// subscriptions gets a note: each subscription's objects need a credential
// of their own (serviceoperator.azure.com/credential-from).

// ASO annotations, and the converter's own.
const (
	AnnASOReconcilePolicy = "serviceoperator.azure.com/reconcile-policy"
	AnnAzureSubscription  = "hfl.dev/azure-subscription"
)

// isASO says whether an apiVersion is ASO's or aso-extensions'.
func isASO(apiVersion string) bool {
	g, _, _ := strings.Cut(apiVersion, "/")
	return strings.HasSuffix(g, ".azure.com") || strings.HasSuffix(g, ".azure.hfl.dev")
}

// asoKind is the kind an ARM type converts to.
type asoKind struct {
	APIVersion, Kind string
	// Owned: the parent is the owner. Not for a resource group (its
	// subscription is the credential's) or tenant-level kinds.
	Owned bool
	// OwnerGK: the owner is named with group and kind too (a role
	// assignment's owner is any resource).
	OwnerGK bool
}

// asoKinds, by lower-case ARM type without the API version. The versions
// are the Azure charts' (docs/azure.md "ASO coverage").
var asoKinds = map[string]asoKind{
	"microsoft.resources/resourcegroups":                                            {"resources.azure.com/v1api20200601", "ResourceGroup", false, false},
	"microsoft.managedidentity/userassignedidentities":                              {"managedidentity.azure.com/v1api20230131", "UserAssignedIdentity", true, false},
	"microsoft.managedidentity/userassignedidentities/federatedidentitycredentials": {"managedidentity.azure.com/v1api20230131", "FederatedIdentityCredential", true, false},
	"microsoft.authorization/roleassignments":                                       {"authorization.azure.com/v20220401", "RoleAssignment", true, true},
	"microsoft.network/virtualnetworks":                                             {"network.azure.com/v20250301", "VirtualNetwork", true, false},
	"microsoft.network/virtualnetworks/subnets":                                     {"network.azure.com/v20250301", "VirtualNetworksSubnet", true, false},
	"microsoft.network/natgateways":                                                 {"network.azure.com/v20250301", "NatGateway", true, false},
	"microsoft.network/publicipaddresses":                                           {"network.azure.com/v20250301", "PublicIPAddress", true, false},
	"microsoft.containerservice/managedclusters":                                    {"containerservice.azure.com/v20260501", "ManagedCluster", true, false},
}

const (
	asoAlias        = "subscription.azure.com/v1api20211001"
	extManagement   = "management.azure.hfl.dev/v20230401"
	extResourcesTag = "resources.azure.hfl.dev/v20250401"
)

// azureContext is what one Azure converter needs from the rest of the
// state: which ARM IDs become objects, and under which names.
type azureContext struct {
	// named maps a lower-case ARM ID to the object it converts to.
	named map[string]asoRef
	// address maps a lower-case ARM ID to the Terraform address that
	// makes it, so a reference orders the objects (a state read from show
	// -json records no dependencies).
	address map[string]string
}

type asoRef struct{ Group, Kind, Name string }

func newAzureContext(rs []*TFResource) *azureContext {
	c := &azureContext{named: map[string]asoRef{}, address: map[string]string{}}
	for _, r := range rs {
		if r.Type != "azapi_resource" {
			continue
		}
		t := armType(str(r.Attrs["type"]))
		var k asoKind
		switch t {
		case "microsoft.management/managementgroups":
			k = asoKind{APIVersion: extManagement, Kind: "ManagementGroup"}
		case "microsoft.subscription/aliases":
			k = asoKind{APIVersion: asoAlias, Kind: "Alias"}
		default:
			var ok bool
			if k, ok = asoKinds[t]; !ok {
				continue
			}
		}
		g, _, _ := strings.Cut(k.APIVersion, "/")
		id := strings.ToLower(str(r.Attrs["id"]))
		c.named[id] = asoRef{g, k.Kind, azureObjectName(r, k.Kind)}
		c.address[id] = r.Address
	}
	return c
}

// armType is "Microsoft.Network/virtualNetworks/subnets@2024-07-01" as
// "microsoft.network/virtualnetworks/subnets".
func armType(t string) string {
	t, _, _ = strings.Cut(t, "@")
	return strings.ToLower(t)
}

// azureObjectName is the object's metadata.name: the Azure name, or for a
// role assignment (a GUID) the module's key for it.
func azureObjectName(r *TFResource, kind string) string {
	if kind == "RoleAssignment" {
		if k := lastIndexKey(r.Address); k != "" {
			return sanitize(k)
		}
		return sanitize(r.Name)
	}
	return sanitize(str(r.Attrs["name"]))
}

// lastIndexKey is the last string instance key in an address
// (module.x["a"].y["cluster_network_contributor"] is
// cluster_network_contributor), or "".
func lastIndexKey(addr string) string {
	i := strings.LastIndex(addr, `["`)
	if i < 0 {
		return ""
	}
	k := addr[i+2:]
	if j := strings.Index(k, `"]`); j >= 0 {
		return k[:j]
	}
	return ""
}

// subscriptionOf is the subscription GUID in an ARM ID, or "".
func subscriptionOf(id string) string {
	l := strings.ToLower(id)
	if !strings.HasPrefix(l, "/subscriptions/") {
		return ""
	}
	s := id[len("/subscriptions/"):]
	s, _, _ = strings.Cut(s, "/")
	return s
}

// isARMID says whether s looks like an ARM resource ID.
func isARMID(s string) bool {
	l := strings.ToLower(s)
	return strings.HasPrefix(l, "/subscriptions/") || strings.HasPrefix(l, "/providers/")
}

func convertAzure(r *TFResource, c *azureContext) ([]*Object, []Finding) {
	objs, f := convertAzureResource(r, c)
	for _, o := range objs {
		c.addRefDeps(o, r)
	}
	return objs, f
}

// addRefDeps adds to o.Deps the addresses of the resources r names: its
// parent and every ARM ID in its body or the object's spec.
func (c *azureContext) addRefDeps(o *Object, r *TFResource) {
	seen := map[string]bool{}
	deps := append([]string(nil), o.Deps...)
	for _, d := range deps {
		seen[d] = true
	}
	add := func(id string) {
		a, ok := c.address[strings.ToLower(id)]
		if !ok || a == o.Source || seen[a] {
			return
		}
		seen[a] = true
		deps = append(deps, a)
	}
	var walk func(v any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			if isARMID(x) {
				add(x)
			}
		case map[string]any:
			for _, val := range x {
				walk(val)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	add(str(r.Attrs["parent_id"]))
	walk(r.Attrs["body"])
	walk(r.Attrs["resource_id"])
	walk(o.Obj["spec"])
	o.Deps = deps
}

func convertAzureResource(r *TFResource, c *azureContext) ([]*Object, []Finding) {
	t := armType(str(r.Attrs["type"]))
	switch r.Type {
	case "azapi_resource":
		switch t {
		case "microsoft.management/managementgroups":
			return convertAzureManagementGroup(r, c)
		case "microsoft.subscription/aliases":
			return convertAzureAlias(r, c)
		}
		k, ok := asoKinds[t]
		if !ok {
			return nil, []Finding{{r.Address, r.Type, "gap", "no ASO mapping in the converter yet for " + str(r.Attrs["type"]) + " (docs/azure.md names the ASO or aso-extensions kind): keep it in Terraform, or add it to azure.go"}}
		}
		return convertAzureGeneric(r, c, k)
	case "azapi_resource_action":
		return convertAzureAction(r, c, t)
	case "azapi_update_resource":
		if t == "microsoft.resources/tags" {
			return convertAzureTags(r, c)
		}
		return nil, []Finding{{r.Address, r.Type, "gap", "an update of " + str(r.Attrs["type"]) + " on a resource Terraform does not create: no ASO kind patches part of a resource"}}
	}
	return nil, []Finding{{r.Address, r.Type, "gap", "no Azure mapping in the converter yet (azurerm and other azapi resources): keep it in Terraform, or add it to azure.go"}}
}

// asoObject is one ASO object with the subscription it lives in.
func asoObject(r *TFResource, apiVersion, kind, name, subscription string, spec map[string]any) *Object {
	ann := map[string]any{}
	if subscription != "" {
		ann[AnnAzureSubscription] = subscription
	}
	return &Object{
		Source: r.Address, Call: r.Call, TFType: r.Type, Kind: kind, Deps: r.Deps,
		Obj: map[string]any{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata":   map[string]any{"name": name, "annotations": ann},
			"spec":       spec,
		},
	}
}

// owner names parent: by the object it converts to, or by ARM ID.
func (c *azureContext) owner(parent string, withGK bool) map[string]any {
	if ref, ok := c.named[strings.ToLower(parent)]; ok {
		o := map[string]any{"name": ref.Name}
		if withGK {
			o["group"], o["kind"] = ref.Group, ref.Kind
		}
		return o
	}
	return map[string]any{"armId": parent}
}

func convertAzureGeneric(r *TFResource, c *azureContext, k asoKind) ([]*Object, []Finding) {
	a := r.Attrs
	id := str(a["id"])
	parent := str(a["parent_id"])
	sub := subscriptionOf(id)
	spec := map[string]any{"azureName": str(a["name"])}
	body, _ := a["body"].(map[string]any)
	for key, v := range body {
		if key == "properties" {
			props, _ := v.(map[string]any)
			for pk, pv := range props {
				spec[pk] = pv
			}
			continue
		}
		spec[key] = v
	}
	if l := str(a["location"]); l != "" {
		spec["location"] = l
	}
	if tags, ok := a["tags"].(map[string]any); ok && len(tags) > 0 {
		spec["tags"] = tags
	}
	if idn := azureIdentity(a["identity"]); idn != nil {
		spec["identity"] = idn
	}
	if k.Owned {
		spec["owner"] = c.owner(parent, k.OwnerGK)
	}
	spec = asoValue(spec, "").(map[string]any)
	if spec == nil {
		spec = map[string]any{}
	}
	return []*Object{asoObject(r, k.APIVersion, k.Kind, azureObjectName(r, k.Kind), sub, spec)}, nil
}

// azureIdentity is azapi's identity block as ASO's identity:
// {type, userAssignedIdentities: [{reference: {armId}}]}.
func azureIdentity(v any) map[string]any {
	for _, i := range list(v) {
		m, _ := i.(map[string]any)
		typ := str(m["type"])
		if typ == "" {
			continue
		}
		out := map[string]any{"type": typ}
		var uai []any
		for _, id := range list(m["identity_ids"]) {
			uai = append(uai, map[string]any{"reference": map[string]any{"armId": str(id)}})
		}
		if len(uai) > 0 {
			out["userAssignedIdentities"] = uai
		}
		return out
	}
	return nil
}

// asoValue drops nulls and empties and turns ARM IDs into ASO references:
// an {id} object is {reference: {armId}}, and a field named ...Id or ...ID
// holding an ARM ID is ...Reference: {armId}.
func asoValue(v any, key string) any {
	switch x := v.(type) {
	case map[string]any:
		if len(x) == 1 {
			if id, ok := x["id"].(string); ok && isARMID(id) {
				return map[string]any{"reference": map[string]any{"armId": id}}
			}
		}
		out := map[string]any{}
		for k, val := range x {
			if s, ok := val.(string); ok && isARMID(s) && k != "armId" {
				if base, ok := cutIDSuffix(k); ok {
					out[base+"Reference"] = map[string]any{"armId": s}
					continue
				}
			}
			if nv := asoValue(val, k); !empty(nv) {
				out[k] = nv
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		var out []any
		for _, e := range x {
			if nv := asoValue(e, key); !empty(nv) {
				out = append(out, nv)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return v
}

// pruneEmpty drops nulls and empties, and changes nothing else.
func pruneEmpty(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			if nv := pruneEmpty(val); !empty(nv) {
				out[k] = nv
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		var out []any
		for _, e := range x {
			if nv := pruneEmpty(e); !empty(nv) {
				out = append(out, nv)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return v
}

// cutIDSuffix is subnetId or nodeSubnetID without the Id: subnet,
// nodeSubnet.
func cutIDSuffix(k string) (string, bool) {
	for _, s := range []string{"Id", "ID"} {
		if strings.HasSuffix(k, s) && len(k) > len(s) {
			return strings.TrimSuffix(k, s), true
		}
	}
	return "", false
}

// convertAzureManagementGroup: aso-extensions' ManagementGroup. The
// tenant's root group is the default parent and is left out.
func convertAzureManagementGroup(r *TFResource, c *azureContext) ([]*Object, []Finding) {
	a := r.Attrs
	body, _ := a["body"].(map[string]any)
	props, _ := body["properties"].(map[string]any)
	spec := map[string]any{"azureName": str(a["name"])}
	if dn := str(props["displayName"]); dn != "" {
		spec["displayName"] = dn
	}
	parent := str(dig(props, "details", "parent", "id"))
	var f []Finding
	if parent != "" && !isTenantRoot(parent) {
		ref := map[string]any{"armId": parent}
		if n, ok := c.named[strings.ToLower(parent)]; ok {
			ref = map[string]any{"group": n.Group, "kind": n.Kind, "name": n.Name}
		}
		spec["details"] = map[string]any{"parent": map[string]any{"reference": ref}}
	}
	return []*Object{asoObject(r, extManagement, "ManagementGroup", azureObjectName(r, "ManagementGroup"), "", spec)}, f
}

// isTenantRoot says whether a management group ID is the tenant root
// group's: the root's name is the tenant ID, a GUID.
func isTenantRoot(id string) bool {
	n := id[strings.LastIndex(id, "/")+1:]
	return len(n) == 36 && strings.Count(n, "-") == 4
}

// convertAzureAlias: ASO's Alias for a subscription the module created.
// The subscription it points at is in the alias's output.
func convertAzureAlias(r *TFResource, c *azureContext) ([]*Object, []Finding) {
	a := r.Attrs
	body, _ := a["body"].(map[string]any)
	props, _ := body["properties"].(map[string]any)
	sub := str(dig(a, "output", "properties", "subscriptionId"))
	// The alias's managementGroupId stays an ID (ASO's Alias has no
	// reference for it), so only nulls and empties go.
	p := pruneEmpty(props)
	spec := map[string]any{"azureName": str(a["name"])}
	if p != nil {
		spec["properties"] = p
	}
	f := []Finding{{r.Address, r.Type, "note", "Alias " + str(a["name"]) + " points at subscription " + sub + ": ASO adopts the alias by name. Its properties are read-only after creation; an edit does not move or rename the subscription"}}
	return []*Object{asoObject(r, asoAlias, "Alias", azureObjectName(r, "Alias"), sub, spec)}, f
}

// convertAzureAction: the actions the Azure Verified Modules run.
func convertAzureAction(r *TFResource, c *azureContext, t string) ([]*Object, []Finding) {
	a := r.Attrs
	rid := str(a["resource_id"])
	action := str(a["action"])
	when := str(a["when"])
	switch {
	case t == "microsoft.management/managementgroups/subscriptions" && strings.EqualFold(str(a["method"]), "PUT"):
		// The placement of a subscription under a management group:
		// .../managementGroups/<mg>/subscriptions/<sub>.
		i := strings.LastIndex(strings.ToLower(rid), "/subscriptions/")
		if i < 0 {
			break
		}
		mg, sub := rid[:i], rid[i+len("/subscriptions/"):]
		spec := map[string]any{"azureName": sub, "owner": map[string]any{"armId": mg}}
		name := sanitize(mg[strings.LastIndex(mg, "/")+1:] + "-" + sub)
		return []*Object{asoObject(r, extManagement, "ManagementGroupsSubscription", name, sub, spec)}, nil
	case strings.HasSuffix(strings.ToLower(action), "/cancel") && when == "destroy":
		return nil, []Finding{{r.Address, r.Type, "terraform-only", "cancels the subscription when Terraform destroys it: not carried. Deleting the converted objects never cancels a subscription (the Alias is detached on delete)"}}
	case strings.HasSuffix(strings.ToLower(action), "/rename") || strings.HasSuffix(strings.ToLower(action), "rename"):
		name := str(dig(a, "body", "subscriptionName"))
		return nil, []Finding{{r.Address, r.Type, "note", "renamed subscription " + subscriptionOf(rid) + " to " + name + " once, with an action: the name stays; no ASO kind renames a subscription"}}
	}
	return nil, []Finding{{r.Address, r.Type, "gap", "an action (" + str(a["method"]) + " " + str(a["type"]) + " " + action + "), not a resource: no KRM object runs it"}}
}

// convertAzureTags: sub-vending's tags on a subscription (an update of
// Microsoft.Resources/tags) are aso-extensions' TagSet, owned by the
// subscription.
func convertAzureTags(r *TFResource, c *azureContext) ([]*Object, []Finding) {
	a := r.Attrs
	parent := str(a["parent_id"])
	tags := dig(a, "body", "properties", "tags")
	spec := map[string]any{"owner": map[string]any{"armId": parent}}
	if m, ok := tags.(map[string]any); ok && len(m) > 0 {
		spec["properties"] = map[string]any{"tags": m}
	}
	sub := subscriptionOf(parent)
	return []*Object{asoObject(r, extResourcesTag, "TagSet", sanitize("tags-"+sub), sub, spec)}, nil
}

// azureSubscriptionNote is the note for objects that span subscriptions.
func azureSubscriptionNote(objs []*Object) []Finding {
	seen := map[string]bool{}
	for _, o := range objs {
		if s := str(dig(o.Obj, "metadata", "annotations", AnnAzureSubscription)); s != "" {
			seen[s] = true
		}
	}
	subs := sortedKeys(seen)
	if len(subs) < 2 {
		return nil
	}
	return []Finding{{"", "", "note", "objects in " + strings.Join(subs, ", ") + ": ASO takes the subscription from its credential, so each subscription's objects need a credential of their own (serviceoperator.azure.com/credential-from); hfl.dev/azure-subscription says which"}}
}
