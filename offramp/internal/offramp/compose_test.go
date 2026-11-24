// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"
)

func needHelm(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("helm"); err != nil {
		t.Skip("helm is not installed: the composed charts cannot be rendered")
	}
}

func composeFixture(t *testing.T, state string) []*Composed {
	t.Helper()
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(state)
	if err != nil {
		t.Fatal(err)
	}
	res := ConvertState(st, k, Options{DeletionPolicy: "abandon"})
	cs, err := ComposeState(res, st, ComposeStateOptions{Out: t.TempDir(), Environment: "prod", StatePath: state})
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

// Every call of every fixture composes into a chart that lints and renders
// its state with no difference. The only collisions are on things outside
// the chart a second instance would really share: a grant or an
// organization policy on a parent folder, a grant on another call's subnet.
func TestComposeFixtures(t *testing.T) {
	needHelm(t)
	sharedOutside := map[string]bool{
		"folders/auditor":              true,
		"folders/require-os-login":     true,
		"folders/resource-locations":   true,
		"folders/skip-default-network": true,
		"live-shared-vpc/root":         true,
	}
	states := map[string]string{
		"gke":              "testdata/synthetic/gke/terraform.tfstate",
		"shared-vpc":       "testdata/synthetic/shared-vpc/terraform.tfstate",
		"platform-project": "testdata/synthetic/platform-project/terraform.tfstate",
		"folders":          "testdata/synthetic/folders/terraform.tfstate",
		"live-shared-vpc":  "testdata/live/shared-vpc.tfstate",
	}
	for shape, state := range states {
		t.Run(shape, func(t *testing.T) {
			cs := composeFixture(t, state)
			if len(cs) == 0 {
				t.Fatal("no charts composed")
			}
			for _, c := range cs {
				if !c.Verify.OK() {
					t.Errorf("%s not proven:\n%s", c.Spec.Chart.Name, VerifyReport(c.Verify))
				}
				key := shape + "/" + c.Spec.Chart.Name
				if len(c.Verify.Collisions) > 0 && !sharedOutside[key] {
					t.Errorf("%s: unexpected collisions: %v", key, c.Verify.Collisions)
				}
				if len(c.Verify.Collisions) == 0 && sharedOutside[key] {
					t.Errorf("%s: expected collisions on a shared parent, found none", key)
				}
			}
		})
	}
}

// The zero-diff check fails when the chart does not render the state: a
// changed default is caught.
func TestVerifyCatchesDifference(t *testing.T) {
	needHelm(t)
	cs := composeFixture(t, "testdata/synthetic/gke/terraform.tfstate")
	c := chart(t, cs, "gke-standard")
	path := filepath.Join(c.Dir, "values.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(b), "appsMachineType: e2-standard-2", "appsMachineType: e2-standard-8", 1)
	if mutated == string(b) {
		t.Fatal("appsMachineType default not found in values.yaml")
	}
	if err := os.WriteFile(path, []byte(mutated), 0o644); err != nil {
		t.Fatal(err)
	}
	objs := objectsOf(t, "testdata/synthetic/gke/terraform.tfstate", "gke_standard")
	v := Verify(c.Spec, c.Dir, objs)
	if v.OK() || len(v.Differences) != 1 || !strings.Contains(v.Differences[0].Path, "machineType") {
		t.Fatalf("want one machineType difference, got %v", v.Differences)
	}
}

// Overriding one placement value moves only what it should: the cluster's
// subnetwork ref follows subnetwork, its network ref follows network (both
// held "ofr-test-gke" in the state), and node pools follow the cluster.
func TestPlacementWiring(t *testing.T) {
	needHelm(t)
	c := chart(t, composeFixture(t, "testdata/synthetic/gke/terraform.tfstate"), "gke-standard")
	objs, err := renderChart(c.Dir, "x", "", map[string]any{"subnetwork": "sub-b", "network": "net-a", "codeName": "kite", "region": "europe-west4"})
	if err != nil {
		t.Fatal(err)
	}
	var cluster, pool map[string]any
	for _, o := range objs {
		switch o["kind"] {
		case "ContainerCluster":
			cluster = o
		case "ContainerNodePool":
			pool = o
		}
	}
	for path, want := range map[string]string{
		"spec.subnetworkRef.external": "projects/ofr-test-project/regions/europe-west4/subnetworks/sub-b",
		"spec.networkRef.external":    "projects/ofr-test-project/global/networks/net-a",
		"spec.location":               "europe-west4-a",
		"metadata.name":               "kite",
	} {
		if got := str(digPath(cluster, path)); got != want {
			t.Errorf("cluster %s = %q, want %q", path, got, want)
		}
	}
	if got := str(dig(pool, "spec", "clusterRef", "name")); got != "kite" {
		t.Errorf("node pool clusterRef.name = %q, want kite", got)
	}
}

// Cloud-assigned IDs are adoption bindings: present by default, gone for a
// new instance.
func TestCloudAssignedIDs(t *testing.T) {
	needHelm(t)
	c := chart(t, composeFixture(t, "testdata/synthetic/folders/terraform.tfstate"), "environments")
	ids, ok := valueDefault(c.Spec, "resourceIds").(map[string]any)
	if !ok || len(ids) == 0 {
		t.Fatal("no resourceIds value for the folders' numbers")
	}
	cleared := map[string]any{}
	for k := range ids {
		cleared[k] = nil
	}
	objs, err := renderChart(c.Dir, "x", "", map[string]any{"resourceIds": cleared})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range objs {
		if o["kind"] == "Folder" && dig(o, "spec", "resourceID") != nil {
			t.Errorf("folder %v keeps resourceID %v for a new instance", dig(o, "metadata", "name"), dig(o, "spec", "resourceID"))
		}
	}
}

// Every spec the fixtures compose validates against the ChartSpec schema,
// and every chart exposes the standard placement block.
func TestChartSpecSchema(t *testing.T) {
	needHelm(t)
	c := jsonschema.NewCompiler()
	sch, err := c.Compile("../../schema/chartspec.v1alpha1.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"testdata/synthetic/gke/terraform.tfstate", "testdata/synthetic/platform-project/terraform.tfstate"} {
		for _, cs := range composeFixture(t, state) {
			b, err := yaml.Marshal(cs.Spec)
			if err != nil {
				t.Fatal(err)
			}
			j, err := yaml.YAMLToJSON(b)
			if err != nil {
				t.Fatal(err)
			}
			var doc any
			if err := json.Unmarshal(j, &doc); err != nil {
				t.Fatal(err)
			}
			if err := sch.Validate(doc); err != nil {
				t.Errorf("%s: %v", cs.Spec.Chart.Name, err)
			}
		}
	}
}

func chart(t *testing.T, cs []*Composed, name string) *Composed {
	t.Helper()
	for _, c := range cs {
		if c.Spec.Chart.Name == name {
			return c
		}
	}
	t.Fatalf("no chart %s", name)
	return nil
}

func valueDefault(s *ChartSpec, name string) any {
	for _, v := range s.Values {
		if v.Name == name {
			return v.Default
		}
	}
	return nil
}

func objectsOf(t *testing.T, state, call string) []*Object {
	t.Helper()
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	st, err := ReadState(state)
	if err != nil {
		t.Fatal(err)
	}
	var out []*Object
	for _, o := range ConvertState(st, k, Options{DeletionPolicy: "abandon"}).Objects {
		if o.Call == call {
			out = append(out, o)
		}
	}
	return out
}

// The names value overrides any resource's name outright, for an
// organization's own convention, and references inside the chart follow.
func TestNameEscapeHatch(t *testing.T) {
	needHelm(t)
	c := chart(t, composeFixture(t, "testdata/synthetic/gke/terraform.tfstate"), "gke-standard")
	var cid string
	for _, r := range c.Spec.Resources {
		if r.Kind == "ContainerCluster" {
			cid = r.ID
		}
	}
	objs, err := renderChart(c.Dir, "x", "", map[string]any{"names": map[string]any{cid: "acme-prod-gke"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range objs {
		switch o["kind"] {
		case "ContainerCluster":
			if n, id := dig(o, "metadata", "name"), dig(o, "spec", "resourceID"); n != "acme-prod-gke" || id != "acme-prod-gke" {
				t.Errorf("cluster name %v resourceID %v, want acme-prod-gke", n, id)
			}
		case "ContainerNodePool":
			if got := dig(o, "spec", "clusterRef", "name"); got != "acme-prod-gke" {
				t.Errorf("node pool %v clusterRef.name = %v, want acme-prod-gke", dig(o, "metadata", "name"), got)
			}
		}
	}
	// The schema names the resources a name can be given for.
	if _, err := renderChart(c.Dir, "x", "", map[string]any{"names": map[string]any{"no-such-resource": "x"}}); err == nil {
		t.Error("a name for an unknown resource was accepted")
	}
}

// A grant on a folder in the chart follows the folder's overridden name.
func TestNameEscapeHatchFollowsReferences(t *testing.T) {
	needHelm(t)
	c := chart(t, composeFixture(t, "testdata/synthetic/folders/terraform.tfstate"), "environments")
	names := map[string]any{}
	for _, r := range c.Spec.Resources {
		if r.Kind == "Folder" {
			names[r.ID] = "acme-" + r.ID
		}
	}
	objs, err := renderChart(c.Dir, "x", "", map[string]any{"names": names})
	if err != nil {
		t.Fatal(err)
	}
	refs := 0
	for _, o := range objs {
		walkStrings(o["spec"], "spec", func(path, s string) {
			if strings.HasSuffix(path, "Ref.name") && dig(o, "spec", "resourceRef", "kind") == "Folder" {
				refs++
				if !strings.HasPrefix(s, "acme-") {
					t.Errorf("%v %s = %q does not follow the folder's name", o["kind"], path, s)
				}
			}
		})
	}
	if refs == 0 {
		t.Error("no references to folders inside the chart")
	}
}
