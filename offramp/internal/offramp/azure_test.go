// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// The fixtures are tf-infra's Azure units as they would be after an apply:
// each unit's read-only plan with the values Azure fills in (IDs, principal
// IDs) and fake subscription and tenant IDs (testdata/azure/README.md).
const vendoredASOExt = "../../../crds/aso-extensions"

// asoCRDs loads the ASO CRDs the converter emits (testdata/aso, one version
// each) and aso-extensions', keyed by apiVersion and kind.
func asoCRDs(t *testing.T) map[string]*CRD {
	t.Helper()
	out := map[string]*CRD{}
	for _, dir := range []string{"testdata/aso", vendoredASOExt} {
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no CRDs in %s: %v", dir, err)
		}
		for _, f := range files {
			c, err := loadCRD(f)
			if err != nil {
				t.Fatalf("%s: %v", f, err)
			}
			if c != nil {
				out[c.APIVersion()+"/"+c.Kind] = c
			}
		}
	}
	return out
}

// schemaErrors lists the fields of v its schema does not have, and the
// required fields it lacks.
func schemaErrors(v any, s *Schema, path string) []string {
	if s == nil {
		return nil
	}
	var errs []string
	switch x := v.(type) {
	case map[string]any:
		for _, r := range s.Required {
			if _, ok := x[r]; !ok {
				errs = append(errs, path+"."+r+": required")
			}
		}
		for k, val := range x {
			ps, ok := s.Properties[k]
			if !ok {
				if s.AdditionalProperties != nil {
					errs = append(errs, schemaErrors(val, s.AdditionalProperties, path+"."+k)...)
					continue
				}
				if len(s.Properties) == 0 && s.Type == "object" {
					continue // free-form
				}
				errs = append(errs, path+"."+k+": not in the schema")
				continue
			}
			errs = append(errs, schemaErrors(val, ps, path+"."+k)...)
		}
	case []any:
		for i, e := range x {
			errs = append(errs, schemaErrors(e, s.Items, path+"["+string(rune('0'+i))+"]")...)
		}
	}
	return errs
}

func convertFixture(t *testing.T, name string, opt Options) *Result {
	t.Helper()
	st, err := ReadState(filepath.Join("testdata", "azure", name+".tfstate"))
	if err != nil {
		t.Fatal(err)
	}
	return ConvertState(st, &KCC{}, opt)
}

func findingsOf(res *Result, class string) []string {
	var out []string
	for _, f := range res.Findings {
		if f.Class == class {
			out = append(out, f.Source+": "+f.Reason)
		}
	}
	sort.Strings(out)
	return out
}

func kindsOf(res *Result) []string {
	var out []string
	for _, o := range res.Objects {
		out = append(out, o.Kind)
	}
	sort.Strings(out)
	return out
}

func objectNamed(res *Result, kind, name string) map[string]any {
	for _, o := range res.Objects {
		if o.Kind == kind && o.Name() == name {
			return o.Obj
		}
	}
	return nil
}

// TestAzureObjectsMatchTheCRDs: every object every fixture converts to is
// valid against its CRD, field for field.
func TestAzureObjectsMatchTheCRDs(t *testing.T) {
	crds := asoCRDs(t)
	for _, fx := range []string{"network", "cluster", "subscriptions"} {
		res := convertFixture(t, fx, Options{})
		if len(res.Objects) == 0 {
			t.Fatalf("%s: no objects", fx)
		}
		for _, o := range res.Objects {
			av := str(o.Obj["apiVersion"])
			c, ok := crds[av+"/"+o.Kind]
			if !ok {
				t.Errorf("%s: %s %s: no CRD for it", fx, av, o.Kind)
				continue
			}
			for _, e := range schemaErrors(o.Obj["spec"], c.Spec, "spec") {
				t.Errorf("%s: %s %s: %s", fx, o.Kind, o.Name(), e)
			}
		}
	}
}

// TestAzureNetwork: tf-infra's network unit (AVM resource group, VNet with
// three subnets, NAT gateway and its public IP, the cluster identity and
// its role on the VNet) converts with no gaps, owners by name and ARM IDs
// as references.
func TestAzureNetwork(t *testing.T) {
	res := convertFixture(t, "network", Options{})
	if g := findingsOf(res, "gap"); len(g) != 0 {
		t.Errorf("gaps: %v", g)
	}
	want := []string{"NatGateway", "PublicIPAddress", "ResourceGroup", "RoleAssignment", "UserAssignedIdentity", "VirtualNetwork", "VirtualNetworksSubnet", "VirtualNetworksSubnet", "VirtualNetworksSubnet"}
	if got := kindsOf(res); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds %v, want %v", got, want)
	}
	rg := "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-nobnt-cus-1"
	for _, c := range []struct {
		kind, name string
		path       []string
		want       any
	}{
		{"ResourceGroup", "rg-nobnt-cus-1", []string{"spec", "location"}, "centralus"},
		{"ResourceGroup", "rg-nobnt-cus-1", []string{"spec", "owner"}, nil},
		{"ResourceGroup", "rg-nobnt-cus-1", []string{"metadata", "annotations", AnnAzureSubscription}, "11111111-1111-1111-1111-111111111111"},
		{"VirtualNetwork", "vnet-nobnt-cus-1", []string{"spec", "owner"}, map[string]any{"name": "rg-nobnt-cus-1"}},
		{"VirtualNetwork", "vnet-nobnt-cus-1", []string{"spec", "addressSpace", "addressPrefixes"}, []any{"10.100.0.0/16"}},
		{"VirtualNetworksSubnet", "snet-nobnt-cus-1-nodes", []string{"spec", "owner"}, map[string]any{"name": "vnet-nobnt-cus-1"}},
		{"VirtualNetworksSubnet", "snet-nobnt-cus-1-nodes", []string{"spec", "natGateway"}, map[string]any{"reference": map[string]any{"armId": rg + "/providers/Microsoft.Network/natGateways/nat-nobnt-cus-1"}}},
		{"VirtualNetworksSubnet", "snet-nobnt-cus-1-apiserver", []string{"spec", "natGateway"}, nil},
		{"VirtualNetworksSubnet", "snet-nobnt-cus-1-apiserver", []string{"spec", "defaultOutboundAccess"}, false},
		{"NatGateway", "nat-nobnt-cus-1", []string{"spec", "publicIpAddresses"}, []any{map[string]any{"reference": map[string]any{"armId": rg + "/providers/Microsoft.Network/publicIPAddresses/pip-nat-nobnt-cus-1"}}}},
		{"RoleAssignment", "cluster-network-contributor", []string{"spec", "owner"}, map[string]any{"group": "network.azure.com", "kind": "VirtualNetwork", "name": "vnet-nobnt-cus-1"}},
		{"RoleAssignment", "cluster-network-contributor", []string{"spec", "roleDefinitionReference", "armId"}, "/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.Authorization/roleDefinitions/4d97b98b-1d4f-4787-a291-c67834d212e7"},
		{"RoleAssignment", "cluster-network-contributor", []string{"spec", "principalType"}, "ServicePrincipal"},
	} {
		o := objectNamed(res, c.kind, c.name)
		if o == nil {
			t.Errorf("no %s %s", c.kind, c.name)
			continue
		}
		if got := dig(o, c.path...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s %s %s = %#v, want %#v", c.kind, c.name, strings.Join(c.path, "."), got, c.want)
		}
	}
	// The role assignment keeps its GUID name, so it adopts the assignment.
	ra := objectNamed(res, "RoleAssignment", "cluster-network-contributor")
	if n := str(dig(ra, "spec", "azureName")); len(n) != 36 {
		t.Errorf("role assignment azureName %q, want the assignment's GUID", n)
	}
}

// TestAzureCluster: the AKS Automatic cluster from the AVM module, with
// its subnets and identity by reference.
func TestAzureCluster(t *testing.T) {
	res := convertFixture(t, "cluster", Options{})
	if g := findingsOf(res, "gap"); len(g) != 0 {
		t.Errorf("gaps: %v", g)
	}
	o := objectNamed(res, "ManagedCluster", "aks-nobnt-cus-1")
	if o == nil {
		t.Fatalf("no ManagedCluster: %v", kindsOf(res))
	}
	for _, c := range []struct {
		path []string
		want any
	}{
		{[]string{"spec", "sku", "name"}, "Automatic"},
		{[]string{"spec", "owner"}, map[string]any{"armId": "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-nobnt-cus-1"}},
		{[]string{"spec", "identity", "type"}, "UserAssigned"},
		{[]string{"spec", "networkProfile", "outboundType"}, "userAssignedNATGateway"},
	} {
		if got := dig(o, c.path...); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %#v, want %#v", strings.Join(c.path, "."), got, c.want)
		}
	}
	if dig(o, "spec", "apiServerAccessProfile", "subnetReference", "armId") == nil {
		t.Errorf("apiServerAccessProfile.subnetId must be subnetReference: %v", dig(o, "spec", "apiServerAccessProfile"))
	}
}

// TestAzureSubscriptions: the tenant layout from the AVM ALZ and
// sub-vending modules. Management groups, aliases, placements, tags and
// the CI identities convert; the rename and the cancel at destroy are
// actions, named in findings, never objects.
func TestAzureSubscriptions(t *testing.T) {
	res := convertFixture(t, "subscriptions", Options{})
	if g := findingsOf(res, "gap"); len(g) != 0 {
		t.Errorf("gaps: %v", g)
	}
	count := map[string]int{}
	for _, k := range kindsOf(res) {
		count[k]++
	}
	want := map[string]int{
		"ManagementGroup": 3, "Alias": 2, "ManagementGroupsSubscription": 3, "TagSet": 3,
		"ResourceGroup": 2, "UserAssignedIdentity": 2, "FederatedIdentityCredential": 2, "RoleAssignment": 2,
	}
	if !reflect.DeepEqual(count, want) {
		t.Errorf("kinds %v, want %v", count, want)
	}
	mg := objectNamed(res, "ManagementGroup", "hfl-dev")
	if mg == nil || dig(mg, "spec", "details") != nil || str(dig(mg, "spec", "displayName")) != "dev" {
		t.Errorf("hfl-dev under the tenant root, named dev: %v", mg)
	}
	fic := objectNamed(res, "FederatedIdentityCredential", "tf-infra-ci-cloud-build")
	if str(dig(fic, "spec", "issuer")) != "https://accounts.google.com" || dig(fic, "spec", "owner", "name") != "id-tf-infra-ci" {
		t.Errorf("federated credential: %v", dig(fic, "spec"))
	}
	var cancels, renames int
	for _, f := range res.Findings {
		switch {
		case strings.Contains(f.Reason, "cancels the subscription"):
			cancels++
			if f.Class != "terraform-only" {
				t.Errorf("cancel at destroy: class %s", f.Class)
			}
		case strings.Contains(f.Reason, "renamed subscription"):
			renames++
		}
	}
	if cancels != 2 || renames != 3 {
		t.Errorf("cancels %d (want 2, one per created subscription), renames %d (want 3)", cancels, renames)
	}
	if n := findingsOf(res, "note"); !anyContains(n, "credential of their own") {
		t.Errorf("objects in three subscriptions need the credential note: %v", n)
	}
	for _, o := range res.Objects {
		if strings.Contains(str(dig(o.Obj, "spec", "properties", "billingScope")), "Billing") && o.Kind != "Alias" {
			t.Errorf("%s carries a billing scope", o.Kind)
		}
	}
}

func anyContains(l []string, s string) bool {
	for _, x := range l {
		if strings.Contains(x, s) {
			return true
		}
	}
	return false
}

// TestAzureReconcilePolicy: abandon (the default) leaves the resource on
// delete, delete lets ASO delete, and skip only reads.
func TestAzureReconcilePolicy(t *testing.T) {
	for _, c := range []struct {
		opt  Options
		want any
	}{
		{Options{}, "detach-on-delete"},
		{Options{DeletionPolicy: "delete"}, nil},
		{Options{AzureReconcilePolicy: "skip"}, "skip"},
	} {
		res := convertFixture(t, "network", c.opt)
		for _, o := range res.Objects {
			ann := o.Obj["metadata"].(map[string]any)["annotations"].(map[string]any)
			if got := ann[AnnASOReconcilePolicy]; !reflect.DeepEqual(got, c.want) {
				t.Errorf("%+v: %s reconcile-policy %v, want %v", c.opt, o.Kind, got, c.want)
			}
			if _, ok := ann[AnnDeletionPolicy]; ok {
				t.Errorf("%s: Config Connector annotations on an Azure object", o.Kind)
			}
		}
	}
}

// TestAzureUnmappedIsAGap: an ARM type with no mapping is a gap naming
// azure.go, and azurerm is a gap too.
func TestAzureUnmappedIsAGap(t *testing.T) {
	st := &State{Version: 4, Resources: []StateResource{
		{Mode: "managed", Type: "azapi_resource", Name: "kv", Provider: `provider["registry.opentofu.org/azure/azapi"]`,
			Instances: []StateInstance{{Attributes: map[string]any{"type": "Microsoft.KeyVault/vaults@2024-11-01", "name": "kv", "id": "/subscriptions/x/resourceGroups/r/providers/Microsoft.KeyVault/vaults/kv"}}}},
		{Mode: "managed", Type: "azurerm_resource_group", Name: "rg", Provider: `provider["registry.terraform.io/hashicorp/azurerm"]`,
			Instances: []StateInstance{{Attributes: map[string]any{"name": "rg"}}}},
	}}
	res := ConvertState(st, &KCC{}, Options{})
	g := findingsOf(res, "gap")
	if len(res.Objects) != 0 || len(g) != 2 || !anyContains(g, "azure.go") {
		t.Errorf("objects %d, gaps %v", len(res.Objects), g)
	}
}

// TestAzureOrderFollowsReferences: a state with no recorded dependencies
// (these fixtures, or show -json) still orders the components, from the
// owners and ARM IDs each object names.
func TestAzureOrderFollowsReferences(t *testing.T) {
	res := convertFixture(t, "network", Options{})
	b := Group(res.Objects, BundtOptions{Name: "az"})
	after := map[string][]string{}
	for _, c := range b.Components {
		after[c.Name] = c.DependsOn
	}
	for name, want := range map[string][]string{
		"resource-group": nil,
		"nat":            {"resource-group"},
		"root":           {"resource-group"},
		"vnet":           {"nat", "resource-group"},
	} {
		if !reflect.DeepEqual(after[name], want) {
			t.Errorf("%s after %v, want %v", name, after[name], want)
		}
	}
	// One subscription's CI resource group is not another's.
	res = convertFixture(t, "subscriptions", Options{})
	for _, o := range res.Objects {
		if o.Kind != "UserAssignedIdentity" {
			continue
		}
		sub := str(dig(o.Obj, "metadata", "annotations", AnnAzureSubscription))
		for _, d := range o.Deps {
			for _, p := range res.Objects {
				if p.Source == d && p.Kind == "ResourceGroup" && str(dig(p.Obj, "metadata", "annotations", AnnAzureSubscription)) != sub {
					t.Errorf("%s in %s depends on the resource group in another subscription", o.Source, sub)
				}
			}
		}
	}
}
