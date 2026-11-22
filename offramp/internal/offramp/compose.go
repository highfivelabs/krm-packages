// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

// Compose is the Terraform flavor of the chart composer (Bundt's "Chart
// Composer" design, BND-62): it turns the Config Connector objects
// converted from a state into a ChartSpec, the composer's intermediate
// representation, which Compile turns into a catalog-shaped chart and
// Verify proves renders to the state with no difference.
//
// The value choice here is deterministic. The standard placement block is
// always exposed; sizing knobs come from a table of known fields per kind.
// An agent may later propose more (BND-61); the rules accept or reject
// them, and the zero-diff check still decides whether the chart ships.

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ChartSpec API version. The schema is schema/chartspec.v1alpha1.json.
const ChartSpecAPIVersion = "composer.bundt.hfl.dev/v1alpha1"

// ChartSpec is what the composer compiles. Agents write or edit a
// ChartSpec, never templates.
type ChartSpec struct {
	APIVersion string        `json:"apiVersion"`
	Kind       string        `json:"kind"`
	Metadata   SpecMeta      `json:"metadata"`
	Source     SpecSource    `json:"source"`
	Chart      SpecChart     `json:"chart"`
	Values     []SpecValue   `json:"values"`
	Resources  []SpecRes     `json:"resources"`
	Outputs    []SpecOutput  `json:"outputs,omitempty"`
	Findings   []SpecFinding `json:"findings,omitempty"`
}

type SpecMeta struct {
	Name string `json:"name"`
}

// SpecSource records where the spec came from, for the chart's provenance.
type SpecSource struct {
	// Kind is terraform-state (later: intent, working-set, chart).
	Kind  string `json:"kind"`
	State string `json:"state,omitempty"`
	// Tool is terraform or opentofu; Form is state or show-json.
	Tool             string   `json:"tool,omitempty"`
	Form             string   `json:"form,omitempty"`
	TerraformVersion string   `json:"terraformVersion,omitempty"`
	Call             string   `json:"call,omitempty"`
	Addresses        []string `json:"addresses,omitempty"`
}

type SpecChart struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

// Value tiers.
const (
	TierPlacement = "placement" // the standard placement block, in every chart
	TierKnob      = "knob"      // a field two real instances would set differently
	TierAdoption  = "adoption"  // how Bundt holds the objects (deletion policy)
)

// SpecValue is one chart value. Default is what the state holds, so the
// chart renders the state with no values given.
type SpecValue struct {
	Name        string `json:"name"`
	Tier        string `json:"tier"`
	Type        string `json:"type"`
	Default     any    `json:"default"`
	Description string `json:"description"`
	// Identity says the value reaches an identity field (a name, a
	// location, a project): changing it recreates the resource.
	Identity bool `json:"identity,omitempty"`
	// Uses counts the fields the value binds; 0 means exposed but unused.
	Uses int `json:"uses"`
}

// SpecRes is one resource: a normalized object whose bound fields hold
// ${values.<name>} (the whole field, typed) or interpolate it in a string.
// A labels map may hold the key "$merge": "${values.labels}".
type SpecRes struct {
	ID             string         `json:"id"`
	Kind           string         `json:"kind"`
	Stateful       bool           `json:"stateful,omitempty"`
	IdentityFields []string       `json:"identityFields,omitempty"`
	Object         map[string]any `json:"object"`
}

// SpecOutput is a CEL expression over resources (by hfl.dev/id) and inputs,
// as a catalog Package declares its outputs.
type SpecOutput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SpecFinding is something the composer could not decide, or a caveat.
type SpecFinding struct {
	Resource string `json:"resource,omitempty"`
	Reason   string `json:"reason"`
}

// ComposeOptions steer Compose.
type ComposeOptions struct {
	Name        string // the chart's name
	Version     string // the chart's version (default 0.1.0)
	Environment string // the environment's literal in names and labels, when known
	Moniker     string // the stack moniker; default: derived from the names
	Source      SpecSource
}

// statefulKinds hold data: their objects always carry deletion-policy
// abandon, whatever the deletionPolicy value says.
var statefulKinds = map[string]bool{
	"SQLInstance": true, "SQLDatabase": true, "StorageBucket": true,
	"SpannerInstance": true, "SpannerDatabase": true, "BigtableInstance": true,
	"BigtableTable": true, "BigQueryDataset": true, "BigQueryTable": true,
	"KMSKeyRing": true, "KMSCryptoKey": true, "ComputeDisk": true,
	"RedisInstance": true, "FilestoreInstance": true, "AlloyDBCluster": true,
	"AlloyDBInstance": true, "SecretManagerSecret": true, "PubSubTopic": false,
	"ArtifactRegistryRepository": true, "FirestoreDatabase": true,
	"LoggingLogBucket": true, "ComputeSnapshot": true,
}

// knobFields are sizing fields worth exposing when present, per kind.
var knobFields = map[string][]string{
	"ContainerCluster":           {"spec.releaseChannel.channel", "spec.minMasterVersion"},
	"ContainerNodePool":          {"spec.nodeConfig.machineType", "spec.nodeConfig.diskSizeGb", "spec.nodeConfig.diskType", "spec.autoscaling.minNodeCount", "spec.autoscaling.maxNodeCount", "spec.autoscaling.totalMinNodeCount", "spec.autoscaling.totalMaxNodeCount", "spec.nodeCount", "spec.version"},
	"ComputeSubnetwork":          {"spec.ipCidrRange"},
	"ComputeInstance":            {"spec.machineType"},
	"ComputeDisk":                {"spec.size", "spec.type"},
	"SQLInstance":                {"spec.settings.tier", "spec.settings.diskSize", "spec.settings.availabilityType"},
	"StorageBucket":              {"spec.storageClass"},
	"RedisInstance":              {"spec.memorySizeGb", "spec.tier"},
	"SpannerInstance":            {"spec.numNodes", "spec.processingUnits"},
	"AlloyDBInstance":            {"spec.machineConfig.cpuCount"},
	"BigtableInstance":           {"spec.cluster"},
	"FilestoreInstance":          {"spec.tier"},
	"ComputeRouterNAT":           {"spec.minPortsPerVm"},
	"ArtifactRegistryRepository": {"spec.format"},
}

// identityPaths are fields whose change recreates the resource, for every
// kind (the Registry will carry these per kind: BND-53).
var identityPaths = []string{"metadata.name", "spec.resourceID", "spec.location", "spec.region", "spec.zone", "spec.projectRef", "metadata.annotations.cnrm.cloud.google.com/project-id"}

const (
	annProject = "cnrm.cloud.google.com/project-id"
	// mergeKey in a labels map merges the labels value into it.
	mergeKey = "$merge"
)

var regionRE = regexp.MustCompile(`^[a-z]+-[a-z]+[0-9]+$`)
var zoneRE = regexp.MustCompile(`^([a-z]+-[a-z]+[0-9]+)-[a-z]$`)

// Compose builds a ChartSpec from one group of converted objects.
func Compose(objs []*Object, opt ComposeOptions) (*ChartSpec, error) {
	if len(objs) == 0 {
		return nil, fmt.Errorf("no objects to compose")
	}
	if opt.Version == "" {
		opt.Version = "0.1.0"
	}
	spec := &ChartSpec{
		APIVersion: ChartSpecAPIVersion,
		Kind:       "ChartSpec",
		Metadata:   SpecMeta{Name: opt.Name},
		Source:     opt.Source,
		Chart: SpecChart{
			Name:        opt.Name,
			Version:     opt.Version,
			Description: fmt.Sprintf("Composed by the Bundt chart composer from the Terraform state's %s.", describeSource(opt.Source)),
		},
	}
	for _, o := range objs {
		spec.Source.Addresses = append(spec.Source.Addresses, o.Source)
	}
	sort.Strings(spec.Source.Addresses)

	// Normalize.
	res := make([]SpecRes, 0, len(objs))
	for _, o := range objs {
		obj := deepCopy(o.Obj).(map[string]any)
		md := obj["metadata"].(map[string]any)
		ann, _ := md["annotations"].(map[string]any)
		delete(ann, AnnAddress)
		id := str(ann[AnnID])
		if id == "" {
			id = strings.ToLower(o.Kind) + "-" + o.Name()
			if ann == nil {
				ann = map[string]any{}
				md["annotations"] = ann
			}
			ann[AnnID] = id
		}
		res = append(res, SpecRes{ID: id, Kind: o.Kind, Stateful: statefulKinds[o.Kind], Object: obj})
	}
	internalRefs(res)

	// Choose values: placement first, from what the objects hold.
	c := &chooser{spec: spec, res: res}
	project := mostCommon(res, func(r SpecRes) string { return str(dig(r.Object, "metadata", "annotations", annProject)) })
	for _, r := range res {
		// A chart that makes the project: the project is its resource ID.
		if r.Kind == "Project" {
			project = orDefault(str(dig(r.Object, "spec", "resourceID")), str(dig(r.Object, "metadata", "name")))
		}
	}
	region := mostCommon(res, func(r SpecRes) string {
		for _, f := range []string{"region", "location", "zone"} {
			s := str(dig(r.Object, "spec", f))
			if regionRE.MatchString(s) {
				return s
			}
			if m := zoneRE.FindStringSubmatch(s); m != nil {
				return m[1]
			}
		}
		return ""
	})
	network, subnet := networkNames(res)
	moniker := opt.Moniker
	if moniker == "" {
		moniker = deriveMoniker(res, project)
	}

	c.add(SpecValue{Name: "projectId", Tier: TierPlacement, Type: "string", Default: project, Identity: true,
		Description: "The project the resources live in."})
	c.add(SpecValue{Name: "region", Tier: TierPlacement, Type: "string", Default: region, Identity: true,
		Description: "The region; zones are the region plus their letter."})
	c.add(SpecValue{Name: "network", Tier: TierPlacement, Type: "string", Default: network, Identity: true,
		Description: "The VPC network's name: the one the chart makes, or the one it attaches to."})
	c.add(SpecValue{Name: "subnetwork", Tier: TierPlacement, Type: "string", Default: subnet, Identity: true,
		Description: "The subnetwork's name: the one the chart makes, or the one it attaches to."})
	c.add(SpecValue{Name: "moniker", Tier: TierPlacement, Type: "string", Default: moniker, Identity: true,
		Description: "The stack's short unique name; every name the chart makes starts with it, so instances in one project do not collide. Bundt mints it per StackInstance."})
	c.add(SpecValue{Name: "environment", Tier: TierPlacement, Type: "string", Default: opt.Environment,
		Description: "The Environment's name, where it appears in names and labels."})
	c.add(SpecValue{Name: "labels", Tier: TierPlacement, Type: "object", Default: commonLabels(res),
		Description: "Labels on every labelled resource, merged under each resource's own."})
	c.add(SpecValue{Name: "deletionPolicy", Tier: TierAdoption, Type: "string", Default: "abandon",
		Description: "deletion-policy of resources that hold no data (stateful ones always abandon)."})

	// Cloud-assigned IDs, then knobs: whole fields, before substitution.
	c.cloudAssigned()
	c.knobs(moniker)

	// Substitute placement literals in every string, most specific first.
	var rules []rule
	if subnet != "" {
		rules = append(rules, rule{value: "subnetwork", lit: subnet, prefix: "subnetworks/", whole: wholeNetwork("ComputeSubnetwork", "subnetwork")})
	}
	if network != "" {
		rules = append(rules, rule{value: "network", lit: network, prefix: "networks/", whole: wholeNetwork("ComputeNetwork", "network")})
	}
	if project != "" {
		rules = append(rules, rule{value: "projectId", lit: project, anywhere: true})
	}
	if region != "" {
		rules = append(rules, rule{value: "region", lit: region, anywhere: true})
	}
	if moniker != "" && len(moniker) >= 3 {
		rules = append(rules, rule{value: "moniker", lit: moniker, anywhere: true})
	}
	if opt.Environment != "" && len(opt.Environment) >= 3 {
		rules = append(rules, rule{value: "environment", lit: opt.Environment, names: true})
	}
	for i := range res {
		r := &res[i]
		r.Object = c.subst(r.Kind, "", r.Object, rules).(map[string]any)
		c.labels(r, objs[i].Labelled)
		c.deletion(r)
		r.IdentityFields = identityOf(r.Object)
	}
	spec.Resources = res
	spec.Outputs = outputsOf(res)

	// Findings.
	for _, v := range spec.Values {
		if v.Tier == TierPlacement && v.Uses == 0 && v.Name != "labels" {
			spec.Findings = append(spec.Findings, SpecFinding{Reason: fmt.Sprintf("value %s is exposed (standard placement block) but nothing in the state uses it", v.Name)})
		}
	}
	if len(moniker) > 8 {
		spec.Findings = append(spec.Findings, SpecFinding{Reason: fmt.Sprintf("moniker %q is longer than 8 characters; Bundt mints shorter ones for new instances", moniker)})
	}
	for _, r := range res {
		for _, f := range []string{"metadata.name", "spec.resourceID"} {
			s, ok := digPath(r.Object, f).(string)
			if ok && s != "" && !strings.Contains(s, "${") {
				spec.Findings = append(spec.Findings, SpecFinding{Resource: r.ID, Reason: fmt.Sprintf("%s %q is fixed: two instances in one project would collide", f, s)})
			}
		}
		walkStrings(r.Object, "", func(path, s string) {
			if strings.HasSuffix(path, ".external") && !strings.Contains(s, "${values.") {
				spec.Findings = append(spec.Findings, SpecFinding{Resource: r.ID, Reason: fmt.Sprintf("%s refers outside the chart to %q: a candidate input wired from an upstream StackInstance", path, s)})
			}
		})
		if v, ok := dig(r.Object, "metadata", "labels").(map[string]any); ok {
			if _, ok := v["goog-terraform-provisioned"]; ok {
				spec.Findings = append(spec.Findings, SpecFinding{Resource: r.ID, Reason: "keeps Terraform's goog-terraform-provisioned label so adoption changes nothing; drop it in a later version"})
			}
		}
	}
	return spec, nil
}

func describeSource(s SpecSource) string {
	switch s.Call {
	case "", "root":
		return "root module"
	}
	return "module call " + s.Call
}

// internalRefs turns references to objects in the chart into name refs, so
// the chart wires its own objects and a renamed instance stays wired.
func internalRefs(res []SpecRes) {
	byKindName := map[string]bool{}
	byName := map[string][]string{}
	// A reference may name an object by its name or its resource ID
	// (folders/123); both resolve to the object's name.
	nameOf := map[string]string{}
	for _, r := range res {
		n := str(dig(r.Object, "metadata", "name"))
		byKindName[r.Kind+"/"+n] = true
		byName[n] = append(byName[n], r.Kind)
		if id := str(dig(r.Object, "spec", "resourceID")); id != "" && id != n {
			byKindName[r.Kind+"/"+id] = true
			byName[id] = append(byName[id], r.Kind)
			nameOf[r.Kind+"/"+id] = n
		}
	}
	var fix func(field string, v any) any
	fix = func(field string, v any) any {
		switch t := v.(type) {
		case map[string]any:
			if ext, ok := t["external"].(string); ok && strings.HasSuffix(field, "Ref") || ok && field == "resourceRef" {
				if kind := refTarget(field, ext, t, byKindName, byName); kind != "" {
					n := lastSegment(normRef(ext))
					if m, ok := nameOf[kind+"/"+n]; ok {
						n = m
					}
					out := map[string]any{"name": n}
					if k, ok := t["kind"]; ok {
						out["kind"] = k
					}
					if a, ok := t["apiVersion"]; ok {
						out["apiVersion"] = a
					}
					return out
				}
				return t
			}
			for k, x := range t {
				t[k] = fix(k, x)
			}
			return t
		case []any:
			for i, x := range t {
				t[i] = fix(field, x)
			}
			return t
		}
		return v
	}
	for i := range res {
		if spec, ok := res[i].Object["spec"].(map[string]any); ok {
			fix("spec", spec)
		}
	}
}

// refTarget names the kind of the chart's object a reference points at, or
// "" when it points outside the chart (or cannot tell).
func refTarget(field, ext string, ref map[string]any, byKindName map[string]bool, byName map[string][]string) string {
	name := lastSegment(normRef(ext))
	if k := str(ref["kind"]); k != "" {
		if byKindName[k+"/"+name] {
			return k
		}
		return ""
	}
	kinds := byName[name]
	if len(kinds) == 0 {
		return ""
	}
	// The field names the kind: networkRef → ComputeNetwork (the shortest
	// kind ending in "network", so not ComputeSubnetwork).
	want := strings.ToLower(strings.TrimSuffix(field, "Ref"))
	best := ""
	for _, k := range kinds {
		if strings.HasSuffix(strings.ToLower(k), want) && (best == "" || len(k) < len(best)) {
			best = k
		}
	}
	if best == "" && len(kinds) == 1 && strings.Contains(ext, "/") == false {
		best = kinds[0]
	}
	// A path names its collection; check it agrees.
	if best != "" && strings.Contains(ext, "/") {
		seg := strings.Split(normRef(ext), "/")
		if len(seg) >= 2 {
			coll := strings.ToLower(seg[len(seg)-2])
			if !strings.HasPrefix(coll, strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(best, "Compute"), "Container"))) {
				return ""
			}
		}
	}
	return best
}

func mostCommon(res []SpecRes, f func(SpecRes) string) string {
	n := map[string]int{}
	for _, r := range res {
		if s := f(r); s != "" {
			n[s]++
		}
	}
	best := ""
	for _, k := range sortedKeys(n) {
		if best == "" || n[k] > n[best] {
			best = k
		}
	}
	return best
}

// networkNames finds the network and subnetwork the chart makes or uses.
func networkNames(res []SpecRes) (string, string) {
	var nets, subs []SpecRes
	for _, r := range res {
		switch r.Kind {
		case "ComputeNetwork":
			nets = append(nets, r)
		case "ComputeSubnetwork":
			subs = append(subs, r)
		}
	}
	pick := func(own []SpecRes, field string) string {
		if len(own) == 1 {
			return str(dig(own[0].Object, "metadata", "name"))
		}
		if len(own) > 1 {
			return "" // several: each name stays fixed or follows the moniker
		}
		return mostCommon(res, func(r SpecRes) string {
			var found string
			walkStrings(r.Object, "", func(path, s string) {
				if found == "" && strings.Contains(path, field+"Ref.") {
					found = lastSegment(normRef(s))
				}
			})
			return found
		})
	}
	return pick(nets, "network"), pick(subs, "subnetwork")
}

// deriveMoniker is the name prefix most names share (at a "-"), or the only
// name when there is one.
func deriveMoniker(res []SpecRes, project string) string {
	var names []string
	for _, r := range res {
		// Grants and enabled APIs (named for the API) name nothing the
		// chart makes.
		if isGrant(r.Kind) || r.Kind == "Service" || r.Kind == "Project" {
			continue
		}
		if n := str(dig(r.Object, "metadata", "name")); n != "" {
			// A name that starts with the project ID: the rest is the
			// chart's own part (<project>-checkout-assets).
			if project != "" && strings.HasPrefix(n, project+"-") {
				n = strings.TrimPrefix(n, project+"-")
			}
			// Names the converter built around a cloud-assigned ID
			// (policy-100000050135-1000, n100000000765) describe this
			// adoption, not the chart.
			if n != project && !cloudIDName.MatchString(n) {
				names = append(names, n)
			}
		}
	}
	if len(names) == 0 {
		return ""
	}
	count := map[string]int{}
	for _, n := range names {
		seen := map[string]bool{}
		parts := strings.Split(n, "-")
		for i := 1; i <= len(parts); i++ {
			p := strings.Join(parts[:i], "-")
			if !seen[p] {
				seen[p] = true
				count[p]++
			}
		}
	}
	best := ""
	for _, p := range sortedKeys(count) {
		if len(p) < 3 || (project != "" && strings.Contains(project, p) && len(p) >= len(project)) {
			continue
		}
		if count[p] < 2 && len(names) > 1 {
			continue
		}
		if best == "" || count[p] > count[best] || count[p] == count[best] && len(p) > len(best) {
			best = p
		}
	}
	if best == "" {
		// No shared prefix: the first resource is the module's main one
		// (the cluster, the bucket); its name becomes the moniker.
		if n := names[0]; len(n) >= 3 && n != project {
			best = n
		}
	}
	return best
}

func commonLabels(res []SpecRes) map[string]any {
	var common map[string]any
	n := 0
	for _, r := range res {
		ls, ok := dig(r.Object, "metadata", "labels").(map[string]any)
		if !ok || len(ls) == 0 {
			continue
		}
		n++
		if common == nil {
			common = map[string]any{}
			for k, v := range ls {
				common[k] = v
			}
			continue
		}
		for k, v := range common {
			if ls[k] != v {
				delete(common, k)
			}
		}
	}
	if n < 2 || common == nil {
		return map[string]any{}
	}
	return common
}

type chooser struct {
	spec *ChartSpec
	res  []SpecRes
}

func (c *chooser) add(v SpecValue) {
	if v.Default == nil {
		v.Default = ""
	}
	c.spec.Values = append(c.spec.Values, v)
}

func (c *chooser) use(name string) {
	for i := range c.spec.Values {
		if c.spec.Values[i].Name == name {
			c.spec.Values[i].Uses++
		}
	}
}

func (c *chooser) value(name string) *SpecValue {
	for i := range c.spec.Values {
		if c.spec.Values[i].Name == name {
			return &c.spec.Values[i]
		}
	}
	return nil
}

// knobs exposes the known sizing fields. A field one resource holds is
// named for the field (machineType); one several hold is named per resource
// (appsMachineType).
func (c *chooser) knobs(moniker string) {
	type hit struct {
		res  int
		path string
	}
	byLeaf := map[string][]hit{}
	for i, r := range c.res {
		for _, p := range knobFields[r.Kind] {
			v := digPath(r.Object, p)
			if v == nil {
				continue
			}
			switch v.(type) {
			case map[string]any, []any:
				continue
			}
			leaf := p[strings.LastIndex(p, ".")+1:]
			byLeaf[leaf] = append(byLeaf[leaf], hit{i, p})
		}
	}
	for _, leaf := range sortedKeys(byLeaf) {
		hits := byLeaf[leaf]
		for _, h := range hits {
			r := &c.res[h.res]
			name := leaf
			if len(hits) > 1 || kindCount(c.res, r.Kind) > 1 {
				name = resourceStem(r, moniker) + strings.ToUpper(leaf[:1]) + leaf[1:]
			}
			if c.value(name) != nil {
				name = name + fmt.Sprint(h.res)
			}
			v := digPath(r.Object, h.path)
			c.add(SpecValue{Name: name, Tier: TierKnob, Type: jsonType(v), Default: v, Uses: 1,
				Description: fmt.Sprintf("%s of %s %s.", strings.TrimPrefix(h.path, "spec."), r.Kind, str(dig(r.Object, "metadata", "name")))})
			setPath(r.Object, h.path, "${values."+name+"}")
		}
	}
}

func resourceStem(r *SpecRes, moniker string) string {
	n := str(dig(r.Object, "metadata", "name"))
	if moniker != "" {
		n = strings.TrimPrefix(strings.TrimPrefix(n, moniker), "-")
	}
	if n == "" {
		n = strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(r.Kind, "Compute"), "Container"))
	}
	return lowerFirst(camel(strings.NewReplacer("-", "_", ".", "_").Replace(n)))
}

// rule substitutes one value's literal in strings.
type rule struct {
	value string
	lit   string
	// prefix: the literal counts only right after it (networks/x).
	prefix string
	// whole: the literal counts as the whole string at these places.
	whole func(kind, path string) bool
	// anywhere: at any word boundary.
	anywhere bool
	// names: at word boundaries in names and label values only.
	names bool
}

func wholeNetwork(kind, field string) func(string, string) bool {
	return func(k, path string) bool {
		if k == kind && (path == "metadata.name" || path == "spec.resourceID") {
			return true
		}
		return strings.HasSuffix(path, "."+field+"Ref.name") || strings.HasSuffix(path, "."+field+"Ref.external") || strings.HasSuffix(path, "."+field)
	}
}

// subst applies the rules to every string leaf.
func (c *chooser) subst(kind, path string, v any, rules []rule) any {
	switch t := v.(type) {
	case map[string]any:
		for k, x := range t {
			if path == "metadata.annotations" && k == AnnID {
				continue
			}
			p := k
			if path != "" {
				p = path + "." + k
			}
			t[k] = c.subst(kind, p, x, rules)
		}
		return t
	case []any:
		for i, x := range t {
			t[i] = c.subst(kind, path, x, rules)
		}
		return t
	case string:
		if (strings.HasPrefix(t, "${values.") || strings.HasPrefix(t, "${resourceIds.")) && strings.HasSuffix(t, "}") {
			return t
		}
		segs := []segment{{lit: t}}
		for _, r := range rules {
			segs = r.apply(kind, path, segs, c)
		}
		return joinSegments(segs)
	}
	return v
}

type segment struct {
	lit string
	ref string
}

func (r rule) apply(kind, path string, segs []segment, c *chooser) []segment {
	if len(segs) == 1 && segs[0].ref == "" && segs[0].lit == r.lit && r.whole != nil && r.whole(kind, path) {
		c.use(r.value)
		return []segment{{ref: r.value}}
	}
	nameish := path == "metadata.name" || path == "spec.resourceID" || strings.HasPrefix(path, "metadata.labels.") || strings.HasSuffix(path, ".name") || strings.HasSuffix(path, "displayName")
	if !r.anywhere && r.prefix == "" && !(r.names && nameish) {
		return segs
	}
	var out []segment
	for _, s := range segs {
		if s.ref != "" {
			out = append(out, s)
			continue
		}
		rest := s.lit
		for {
			i := r.find(rest, nameish)
			if i < 0 {
				out = append(out, segment{lit: rest})
				break
			}
			if i > 0 {
				out = append(out, segment{lit: rest[:i]})
			}
			out = append(out, segment{ref: r.value})
			c.use(r.value)
			rest = rest[i+len(r.lit):]
		}
	}
	return compact(out)
}

func (r rule) find(s string, nameish bool) int {
	from := 0
	for {
		j := strings.Index(s[from:], r.lit)
		if j < 0 {
			return -1
		}
		i := from + j
		end := i + len(r.lit)
		okB := (i == 0 || !alnum(s[i-1])) && (end == len(s) || !alnum(s[end]))
		if r.prefix != "" && !r.anywhere {
			before := s[:i]
			okB = okB && (before == r.prefix || strings.HasSuffix(before, "/"+r.prefix))
		}
		if r.names && !r.anywhere && !nameish {
			okB = false
		}
		if okB {
			return i
		}
		from = i + 1
	}
}

func alnum(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

func compact(segs []segment) []segment {
	var out []segment
	for _, s := range segs {
		if s.ref == "" && s.lit == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func joinSegments(segs []segment) string {
	var b strings.Builder
	for _, s := range segs {
		if s.ref != "" {
			b.WriteString("${values." + s.ref + "}")
		} else {
			b.WriteString(s.lit)
		}
	}
	return b.String()
}

// labels moves the common labels out of a labelled resource's own and merges
// the labels value in.
func (c *chooser) labels(r *SpecRes, labelled bool) {
	md := r.Object["metadata"].(map[string]any)
	ls, _ := md["labels"].(map[string]any)
	if !labelled && len(ls) == 0 {
		return
	}
	if ls == nil {
		ls = map[string]any{}
	}
	common, _ := c.value("labels").Default.(map[string]any)
	for k := range common {
		delete(ls, k)
	}
	ls[mergeKey] = "${values.labels}"
	md["labels"] = ls
	c.use("labels")
}

func (c *chooser) deletion(r *SpecRes) {
	ann := r.Object["metadata"].(map[string]any)["annotations"].(map[string]any)
	if r.Stateful {
		ann[AnnDeletionPolicy] = "abandon"
		return
	}
	ann[AnnDeletionPolicy] = "${values.deletionPolicy}"
	c.use("deletionPolicy")
}

func identityOf(obj map[string]any) []string {
	var out []string
	for _, p := range identityPaths {
		if digPath(obj, p) != nil {
			out = append(out, p)
		}
	}
	return out
}

// outputsOf declares each resource's name, and a network's and a
// cluster's usual handles, as CEL over the rendered objects.
func outputsOf(res []SpecRes) []SpecOutput {
	var out []SpecOutput
	seen := map[string]bool{}
	for _, r := range res {
		if isGrant(r.Kind) {
			continue
		}
		n := lowerFirst(camel(strings.NewReplacer("-", "_").Replace(r.ID))) + "Name"
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, SpecOutput{Name: n, Value: fmt.Sprintf("resources[%q].metadata.name", r.ID)})
	}
	return out
}

// digPath reads a dotted path; annotation keys may hold dots, so
// metadata.annotations.<key> takes the rest as the key.
func digPath(m map[string]any, p string) any {
	if strings.HasPrefix(p, "metadata.annotations.") {
		return dig(m, "metadata", "annotations", strings.TrimPrefix(p, "metadata.annotations."))
	}
	return dig(m, strings.Split(p, ".")...)
}

func setPath(m map[string]any, p string, v any) {
	parts := strings.Split(p, ".")
	for _, k := range parts[:len(parts)-1] {
		n, _ := m[k].(map[string]any)
		if n == nil {
			return
		}
		m = n
	}
	m[parts[len(parts)-1]] = v
}

func walkStrings(v any, path string, f func(path, s string)) {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(t) {
			p := k
			if path != "" {
				p = path + "." + k
			}
			walkStrings(t[k], p, f)
		}
	case []any:
		for _, x := range t {
			walkStrings(x, path, f)
		}
	case string:
		f(path, t)
	}
}

func jsonType(v any) string {
	switch v.(type) {
	case bool:
		return "boolean"
	case int, int64, float64:
		if f, ok := v.(float64); ok && f != float64(int64(f)) {
			return "number"
		}
		return "integer"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	}
	return "string"
}

func kindCount(res []SpecRes, kind string) int {
	n := 0
	for _, r := range res {
		if r.Kind == kind {
			n++
		}
	}
	return n
}

// isGrant is an IAM kind that grants rather than makes a resource.
func isGrant(kind string) bool {
	switch kind {
	case "IAMPolicyMember", "IAMPartialPolicy", "IAMPolicy", "IAMBinding", "IAMAuditConfig":
		return true
	}
	return false
}

var cloudIDName = regexp.MustCompile(`(^|-)n?[0-9]{10,}(-|$)`)

var digitsRE = regexp.MustCompile(`^[0-9]+$`)

// cloudAssigned moves resource IDs the cloud assigned (a folder's number, a
// hierarchical firewall policy's ID) into the resourceIds adoption value:
// they bind this adoption to these resources and are empty for a new
// instance, where the cloud assigns new ones.
func (c *chooser) cloudAssigned() {
	ids := map[string]any{}
	for i := range c.res {
		r := &c.res[i]
		s := str(dig(r.Object, "spec", "resourceID"))
		if !digitsRE.MatchString(s) {
			continue
		}
		ids[r.ID] = s
		setPath(r.Object, "spec.resourceID", "${resourceIds."+r.ID+"}")
	}
	if len(ids) == 0 {
		return
	}
	c.add(SpecValue{Name: "resourceIds", Tier: TierAdoption, Type: "object", Default: ids, Uses: len(ids),
		Description: "Cloud-assigned IDs of resources this instance adopts, by resource. Leave empty for a new instance: the cloud assigns them."})
}
