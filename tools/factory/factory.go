// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

// Package factory is the factory generator (docs/design.md, "Factories,
// worked"): it reads a Cloud Foundation Fabric project-factory tree (the
// folders, the project files, the factory's defaults and context) and
// writes the Bundt documents that replace it: one Stack and one Instance
// for the folders, and one Stack and one Instance per project file, each
// Stack listing its components inline from the catalog's charts
// (gcp-folder, gcp-project, gcp-iam-service-account, gcp-gcs).
//
// It maps what the design's table maps and refuses everything else by
// name: a key fabric's factory composes into a module the generator does
// not map yet, a value a composed chart refuses, a lookup it cannot
// resolve, an IAM entry that would make a cycle. A refusal anywhere means
// no documents at all: a partial tree, submitted as a Team-scoped plan,
// would delete what it left out.
package factory

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/highfivelabs/catalog/tools/convert"
)

// Options for Generate.
type Options struct {
	Repo *convert.Repo
	// Config is the generator's config file: the factory's other inputs
	// (defaults, overrides, merges, context) and where its folders and
	// projects are, relative to the file's directory (the factory root).
	Config string
	// FolderIDs are the IDs (folders/N) of the folders this factory makes,
	// keyed by their path in the tree (team-a/app-0): the folders
	// Instance's outputs, read once it is applied. Projects take them as
	// literals; Bundt cannot read another Instance's outputs yet.
	FolderIDs map[string]string
	// Commit is the factory repository's commit, for hfl.dev/factory.
	Commit string
	// Team owns the Stacks; Environment is the Instances' Environment.
	Team, Environment string
	// FoldersOnly writes the folders' documents alone: the first pass,
	// before the folders exist.
	FoldersOnly bool
}

// File is one output file: a Stack and its Instance.
type File struct {
	Name    string // folders.yaml, <project>.yaml
	Content string
}

// Refusal is one thing the generator cannot carry.
type Refusal struct {
	Where string // data/projects/x.yaml: buckets.b.rpo
	Text  string
}

// Error is every refusal of one run.
type Error struct{ Refusals []Refusal }

func (e *Error) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d refused (no documents written):", len(e.Refusals))
	for _, r := range e.Refusals {
		fmt.Fprintf(&b, "\n  %s: %s", r.Where, r.Text)
	}
	return b.String()
}

// The charts the generator composes.
const (
	chartFolder  = "gcp/folder"
	chartProject = "gcp/project"
	chartSA      = "gcp/iam-service-account"
	chartGCS     = "gcp/gcs"
)

// What fabric's factory makes with a module the generator does not map
// yet, and the reason given for each.
var notMapped = map[string]string{
	"automation":          "fabric makes the automation service accounts and state bucket in another project (projects' automation); the generator does not map them yet",
	"billing_budgets":     "budgets come from the factory's budgets folder (billing-account module); the generator does not map them yet",
	"datasets":            "BigQuery datasets (bigquery-dataset module); the generator does not map them yet",
	"dns_threat_detector": "google_network_security_dns_threat_detector has no stable kind (config-connector-extensions' NetworkSecurityDNSThreatDetector) and the project chart has no dns_threat_detector value",
	"kms":                 "KMS key rings and autokeys (kms module); the generator does not map them yet",
	"log_buckets":         "log buckets (logging-bucket module); the generator does not map them yet",
	"project_template":    "project templates (the factory's project-templates folder); the generator does not read them yet",
	"pubsub_topics":       "Pub/Sub topics (pubsub module); the generator does not map them yet",
	"factories_config":    "a project's own factory files (org policies, custom roles, quotas, tags, observability...); the generator does not read them yet, and the chart takes their entries as values",
}

// The keys of a fabric service_accounts.<n> entry.
var saKeys = map[string]bool{
	"description": true, "display_name": true, "iam_self_roles": true, "tag_bindings": true,
	"iam": true, "iam_bindings": true, "iam_bindings_additive": true,
	"iam_billing_bindings": true, "iam_billing_roles": true,
	"iam_folder_bindings": true, "iam_folder_roles": true,
	"iam_organization_bindings": true, "iam_organization_roles": true,
	"iam_project_bindings": true, "iam_project_roles": true,
	"iam_sa_bindings": true, "iam_sa_roles": true,
	"iam_storage_bindings": true, "iam_storage_roles": true,
}

// The keys of the generator's config, and of its defaults, overrides and
// merges (fabric's data_defaults, data_overrides and data_merges, the
// parts the generator maps).
var (
	configKeys   = []string{"defaults", "overrides", "merges", "context", "folders", "projects"}
	defaultsKeys = []string{"billing_account", "bucket", "contacts", "deletion_policy", "labels", "locations", "metric_scopes", "parent", "prefix", "services", "tag_bindings"}
	overrideKeys = []string{"billing_account", "bucket", "contacts", "deletion_policy", "locations", "parent", "prefix", "services", "tag_bindings"}
	mergesKeys   = []string{"contacts", "labels", "metric_scopes", "services", "tag_bindings"}
)

type component struct {
	name   string
	chart  *convert.Chart
	inputs convert.Ordered
	keep   bool
}

type unit struct {
	name       string // the Stack's and the Instance's name
	source     string // the file or folder, relative to the factory root
	components []*component
}

type projectFile struct {
	key           string // basename
	rel           string // relative to the factory root
	data          convert.Ordered
	defaultParent string // $folder_ids:<dir> for a project in the folders tree
}

type gen struct {
	o        Options
	root     string // the factory root (the config file's directory)
	charts   map[string]*convert.Chart
	agents   []serviceAgent
	defaults convert.Ordered
	override convert.Ordered
	merges   convert.Ordered
	context  convert.Ordered
	folders  map[string]convert.Ordered // path in the tree -> .config.yaml
	foldRel  string
	projects map[string]*projectFile
	refused  []Refusal
}

func (g *gen) refuse(where, format string, a ...any) {
	g.refused = append(g.refused, Refusal{Where: where, Text: fmt.Sprintf(format, a...)})
}

// Generate reads the tree and returns the files to write, or an *Error
// listing every refusal.
func Generate(o Options) ([]File, error) {
	if o.Team == "" {
		o.Team = "gcp-factory"
	}
	if o.Environment == "" {
		o.Environment = "gcp-org"
	}
	if o.Commit == "" {
		return nil, fmt.Errorf("no commit: the hfl.dev/factory annotation names the factory's file and commit")
	}
	g := &gen{o: o, root: filepath.Dir(o.Config), charts: map[string]*convert.Chart{}, projects: map[string]*projectFile{}}
	for _, name := range []string{chartFolder, chartProject, chartSA, chartGCS} {
		c, err := o.Repo.ChartByName(name)
		if err != nil {
			return nil, err
		}
		g.charts[name] = c
	}
	agents, err := loadServiceAgents()
	if err != nil {
		return nil, err
	}
	g.agents = agents
	if err := g.loadConfig(); err != nil {
		return nil, err
	}
	if len(g.refused) > 0 {
		return nil, &Error{Refusals: g.refused}
	}

	var units []*unit
	if u := g.folderUnit(); u != nil {
		units = append(units, u)
	}
	if !o.FoldersOnly {
		keys := make([]string, 0, len(g.projects))
		for k := range g.projects {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if u := g.projectUnit(g.projects[k]); u != nil {
				units = append(units, u)
			}
		}
	}
	for _, u := range units {
		g.checkUnit(u)
	}
	if len(g.refused) > 0 {
		return nil, &Error{Refusals: g.refused}
	}
	var files []File
	for _, u := range units {
		files = append(files, File{Name: u.name + ".yaml", Content: g.render(u)})
	}
	return files, nil
}

// ---- reading -------------------------------------------------------------

func readYAML(file string) (convert.Ordered, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var n yaml.Node
	if err := yaml.Unmarshal(data, &n); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if len(n.Content) == 0 {
		return convert.Ordered{}, nil
	}
	v, err := decode(n.Content[0])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	o, ok := v.(convert.Ordered)
	if !ok {
		return nil, fmt.Errorf("%s: not a map", file)
	}
	return o, nil
}

func decode(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		return decode(n.Content[0])
	case yaml.AliasNode:
		return decode(n.Alias)
	case yaml.MappingNode:
		o := convert.Ordered{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == "<<" {
				return nil, fmt.Errorf("line %d: YAML merge keys are not read", n.Content[i].Line)
			}
			v, err := decode(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			o = append(o, convert.Pair{Key: n.Content[i].Value, Value: v})
		}
		return o, nil
	case yaml.SequenceNode:
		l := []any{}
		for _, c := range n.Content {
			v, err := decode(c)
			if err != nil {
				return nil, err
			}
			l = append(l, v)
		}
		return l, nil
	}
	var v any
	if err := n.Decode(&v); err != nil {
		return nil, err
	}
	if i, ok := v.(int); ok {
		return int64(i), nil
	}
	return v, nil
}

func get(o convert.Ordered, k string) (any, bool) {
	for _, p := range o {
		if p.Key == k {
			return p.Value, true
		}
	}
	return nil, false
}

func getMap(o convert.Ordered, k string) convert.Ordered {
	v, _ := get(o, k)
	m, _ := v.(convert.Ordered)
	return m
}

func set(o *convert.Ordered, k string, v any) {
	for i, p := range *o {
		if p.Key == k {
			(*o)[i].Value = v
			return
		}
	}
	*o = append(*o, convert.Pair{Key: k, Value: v})
}

func checkKeys(g *gen, o convert.Ordered, allowed []string, where string) {
	for _, p := range o {
		if !contains(allowed, p.Key) {
			g.refuse(where, "%s is not read by the generator (it reads %s)", p.Key, strings.Join(allowed, ", "))
		}
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func (g *gen) loadConfig() error {
	cfg, err := readYAML(g.o.Config)
	if err != nil {
		return err
	}
	where := filepath.Base(g.o.Config)
	checkKeys(g, cfg, configKeys, where)
	g.defaults = getMap(cfg, "defaults")
	g.override = getMap(cfg, "overrides")
	g.merges = getMap(cfg, "merges")
	g.context = getMap(cfg, "context")
	checkKeys(g, g.defaults, defaultsKeys, where+": defaults")
	checkKeys(g, g.override, overrideKeys, where+": overrides")
	checkKeys(g, g.merges, mergesKeys, where+": merges")
	for _, m := range []struct {
		o    convert.Ordered
		name string
	}{{g.defaults, "defaults"}, {g.override, "overrides"}} {
		checkKeys(g, getMap(m.o, "bucket"), []string{"force_destroy"}, where+": "+m.name+".bucket")
		checkKeys(g, getMap(m.o, "locations"), []string{"storage"}, where+": "+m.name+".locations")
	}

	foldersPath, projectsPath := "folders", "projects"
	if v, ok := get(cfg, "folders"); ok {
		foldersPath, _ = v.(string)
	}
	if v, ok := get(cfg, "projects"); ok {
		projectsPath, _ = v.(string)
	}
	g.foldRel = foldersPath
	g.folders = map[string]convert.Ordered{}
	if err := g.walk(foldersPath, func(rel, file string) error {
		dir := path.Dir(rel)
		if path.Base(rel) == ".config.yaml" {
			if dir == "." {
				g.refuse(path.Join(foldersPath, rel), "a .config.yaml at the top of the folders tree makes no folder")
				return nil
			}
			data, err := readYAML(file)
			if err != nil {
				return err
			}
			g.folders[dir] = data
			return nil
		}
		parent := ""
		if dir != "." {
			parent = "$folder_ids:" + dir
		}
		return g.addProject(path.Join(foldersPath, rel), file, parent)
	}); err != nil {
		return err
	}
	return g.walk(projectsPath, func(rel, file string) error {
		if strings.HasSuffix(rel, ".config.yaml") {
			return nil
		}
		return g.addProject(path.Join(projectsPath, rel), file, "")
	})
}

// walk calls fn for every .yaml file under dir (relative to the factory
// root), in lexical order; a missing dir is empty.
func (g *gen) walk(dir string, fn func(rel, file string) error) error {
	base := filepath.Join(g.root, filepath.FromSlash(dir))
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return err
		}
		rel, err := filepath.Rel(base, p)
		if err != nil {
			return err
		}
		return fn(filepath.ToSlash(rel), p)
	})
}

func (g *gen) addProject(rel, file, parent string) error {
	data, err := readYAML(file)
	if err != nil {
		return err
	}
	key := strings.TrimSuffix(path.Base(rel), ".yaml")
	if prev, ok := g.projects[key]; ok {
		g.refuse(rel, "project %s is also %s: fabric keys projects by file name", key, prev.rel)
		return nil
	}
	g.projects[key] = &projectFile{key: key, rel: rel, data: data, defaultParent: parent}
	return nil
}

// ---- values and lookups ----------------------------------------------------

// first returns the first value set (present and not null).
func first(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}

func val(o convert.Ordered, k string) any {
	v, _ := get(o, k)
	return v
}

// projectName and projectPrefix are fabric's (projects-defaults.tf).
func (g *gen) projectName(p *projectFile) string {
	if s, ok := val(p.data, "name").(string); ok && s != "" {
		return s
	}
	return p.key
}

func (g *gen) projectPrefix(p *projectFile) any {
	if v := val(g.override, "prefix"); v != nil {
		return v
	}
	if v, ok := get(p.data, "prefix"); ok {
		return v
	}
	return val(g.defaults, "prefix")
}

// projectID is the ID the project chart gives a project (its project_id
// output), known from its inputs.
func (g *gen) projectID(p *projectFile) string {
	if s, ok := g.projectPrefix(p).(string); ok && s != "" {
		return s + "-" + g.projectName(p)
	}
	return g.projectName(p)
}

type scope struct {
	project *projectFile    // nil for the folders
	sas     map[string]bool // the project's own service accounts
	where   string
}

var (
	lookup     = regexp.MustCompile(`^\$([a-z_]+):(.+)$`)
	saRef      = regexp.MustCompile(`^(service_accounts|service_agents)/([^/]+)/(.+)$`)
	ownSAExpr  = regexp.MustCompile(`^\$\{components\.sa-(.+)\.outputs\.iam_email\}$`)
	exprRef    = regexp.MustCompile(`\$\{components\.([a-z0-9-]+)\.`)
	identifier = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// resolve turns one fabric value into Bundt's: a $<namespace>:<key>
// lookup becomes an expression on a component of the same Instance or a
// literal, anything else stays. ok is false when it was refused.
func (g *gen) resolve(s string, sc *scope, where string) (string, bool) {
	if strings.Contains(s, "${") {
		g.refuse(where, "%q holds ${...}: Bundt would read it as an expression (fabric's condition_vars templating is not mapped)", s)
		return "", false
	}
	if !strings.HasPrefix(s, "$") {
		return s, true
	}
	m := lookup.FindStringSubmatch(s)
	if m == nil {
		g.refuse(where, "%q is not a fabric lookup ($<namespace>:<key>)", s)
		return "", false
	}
	ns, key := m[1], m[2]
	switch ns {
	case "iam_principals":
		if r := saRef.FindStringSubmatch(key); r != nil {
			return g.resolvePrincipal(r[1], r[2], r[3], s, sc, where)
		}
	case "folder_ids":
		if _, ok := g.folders[key]; ok {
			if sc.project == nil {
				return "${components." + folderComponent(key) + ".outputs.id}", true
			}
			id, ok := g.o.FolderIDs[key]
			if !ok {
				g.refuse(where, "%s: the folder %s is made by the folders Instance and has no ID here: apply that Instance, then pass its folders' IDs (-folder-ids)", s, key)
				return "", false
			}
			return id, true
		}
	case "project_ids":
		if p, ok := g.projects[key]; ok {
			return g.projectID(p), true
		}
	case "project_numbers":
		if _, ok := g.projects[key]; ok {
			g.refuse(where, "%s: a project's number is known only once the project is made, and another Instance's outputs cannot be read yet (cross-Instance references, docs/design.md)", s)
			return "", false
		}
	}
	if v, ok := val(getMap(g.context, ns), key).(string); ok {
		return v, true
	}
	g.refuse(where, "%s: not made by this factory and not in the factory's context (context.%s)", s, ns)
	return "", false
}

func (g *gen) resolvePrincipal(kind, proj, name, s string, sc *scope, where string) (string, bool) {
	self := proj == "_self_" || (sc.project != nil && proj == sc.project.key)
	if self && sc.project == nil {
		g.refuse(where, "%s: _self_ names a project, and a folder has none", s)
		return "", false
	}
	if kind == "service_accounts" {
		if self {
			if !sc.sas[name] {
				g.refuse(where, "%s: the project has no service account %s", s, name)
				return "", false
			}
			return "${components.sa-" + name + ".outputs.iam_email}", true
		}
		p, ok := g.projects[proj]
		if !ok {
			g.refuse(where, "%s: no project %s in this factory", s, proj)
			return "", false
		}
		if _, ok := get(getMap(p.data, "service_accounts"), name); !ok {
			g.refuse(where, "%s: the project %s has no service account %s", s, proj, name)
			return "", false
		}
		// Another Instance's account: its email is known from its inputs.
		return "serviceAccount:" + name + "@" + g.projectID(p) + ".iam.gserviceaccount.com", true
	}
	if !self {
		g.refuse(where, "%s: another project's service agent needs that project's number, known only once it is made, and another Instance's outputs cannot be read yet", s)
		return "", false
	}
	for _, a := range g.agents {
		if a.Name == name || contains(a.Aliases, name) {
			id := strings.ReplaceAll(a.Identity, "${universe_domain}", "")
			id = strings.ReplaceAll(id, "${project_number}", "${components.project.outputs.number}")
			return "serviceAccount:" + id, true
		}
	}
	g.refuse(where, "%s: no service agent %s in fabric's service-agents.yaml", s, name)
	return "", false
}

// resolveAll resolves every string and map key in v.
func (g *gen) resolveAll(v any, sc *scope, where string) any {
	switch x := v.(type) {
	case string:
		s, _ := g.resolve(x, sc, where)
		return s
	case convert.Ordered:
		out := convert.Ordered{}
		for _, p := range x {
			k, _ := g.resolve(p.Key, sc, where+"."+p.Key)
			out = append(out, convert.Pair{Key: k, Value: g.resolveAll(p.Value, sc, where+"."+p.Key)})
		}
		return out
	case []any:
		out := []any{}
		for i, e := range x {
			out = append(out, g.resolveAll(e, sc, fmt.Sprintf("%s[%d]", where, i)))
		}
		return out
	}
	return v
}

func isEmpty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case convert.Ordered:
		return len(x) == 0
	case []any:
		return len(x) == 0
	case string:
		return x == ""
	}
	return false
}

func folderComponent(key string) string { return "folder-" + strings.ReplaceAll(key, "/", "-") }

// ---- folders -------------------------------------------------------------

func (g *gen) folderUnit() *unit {
	if len(g.folders) == 0 {
		return nil
	}
	keys := make([]string, 0, len(g.folders))
	for k := range g.folders {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	u := &unit{name: "folders", source: g.foldRel}
	seen := map[string]string{}
	sc := &scope{}
	for _, k := range keys {
		data := g.folders[k]
		where := path.Join(g.foldRel, k, ".config.yaml")
		cn := folderComponent(k)
		if prev, ok := seen[cn]; ok {
			g.refuse(where, "its component %s is also %s's", cn, prev)
			continue
		}
		seen[cn] = k
		if strings.Count(k, "/") >= 4 {
			g.refuse(where, "fabric's factory makes four levels of folders, and this is deeper")
			continue
		}
		in := convert.Ordered{}
		if id, ok := get(data, "id"); ok && id != nil {
			set(&in, "id", g.resolveAll(id, sc, where+": id"))
			set(&in, "folder_create", false)
		} else {
			set(&in, "name", val(data, "name"))
		}
		parent := val(data, "parent")
		if parent == nil {
			if dir := path.Dir(k); dir != "." {
				if _, ok := g.folders[dir]; !ok {
					g.refuse(where, "its parent directory %s has no .config.yaml", dir)
					continue
				}
				parent = "$folder_ids:" + dir
			} else {
				parent = "$folder_ids:default"
			}
		}
		if _, ok := get(in, "id"); !ok {
			set(&in, "parent", g.resolveAll(parent, sc, where+": parent"))
		}
		for _, p := range data {
			w := where + ": " + p.Key
			switch p.Key {
			case "id", "name", "parent":
			case "automation", "factories_config":
				g.refuse(w, "%s", notMapped[p.Key])
			case "data_access_logs":
				set(&in, "logging_data_access", g.resolveAll(p.Value, sc, w))
			case "logging":
				l, _ := p.Value.(convert.Ordered)
				for _, q := range l {
					if q.Key == "sinks" {
						set(&in, "logging_sinks", g.resolveAll(q.Value, sc, w+".sinks"))
					} else {
						ls := getMap(in, "logging_settings")
						set(&ls, q.Key, g.resolveAll(q.Value, sc, w+"."+q.Key))
						set(&in, "logging_settings", ls)
					}
				}
			default:
				set(&in, p.Key, g.resolveAll(p.Value, sc, w))
			}
		}
		u.components = append(u.components, &component{name: cn, chart: g.charts[chartFolder], inputs: in})
	}
	return u
}

// ---- projects -------------------------------------------------------------

// The project chart's inputs the generator computes from fabric's
// defaults, overrides and merges, in the order they are written when the
// file does not set them.
var computed = []string{"labels", "services", "contacts", "metric_scopes", "tag_bindings"}

func (g *gen) effective(p *projectFile, k string) any {
	file := val(p.data, k)
	switch k {
	case "billing_account", "deletion_policy", "parent":
		if k == "parent" && file == nil && p.defaultParent != "" {
			file = p.defaultParent
		}
		return first(val(g.override, k), file, val(g.defaults, k))
	case "labels", "metric_scopes":
		return mergeValue(first(file, val(g.defaults, k)), val(g.merges, k))
	case "services", "contacts", "tag_bindings":
		return mergeValue(first(val(g.override, k), file, val(g.defaults, k)), val(g.merges, k))
	}
	return file
}

// mergeValue is fabric's data_merges: maps merged (the merge wins), lists
// concatenated without repeats.
func mergeValue(v, m any) any {
	switch mx := m.(type) {
	case convert.Ordered:
		out := convert.Ordered{}
		if vx, ok := v.(convert.Ordered); ok {
			out = append(out, vx...)
		}
		for _, p := range mx {
			set(&out, p.Key, p.Value)
		}
		return out
	case []any:
		var out []any
		if vx, ok := v.([]any); ok {
			out = append(out, vx...)
		}
		for _, e := range mx {
			if !containsAny(out, e) {
				out = append(out, e)
			}
		}
		return out
	}
	return v
}

func containsAny(l []any, v any) bool {
	for _, e := range l {
		if e == v {
			return true
		}
	}
	return false
}

func (g *gen) projectUnit(p *projectFile) *unit {
	where := p.rel
	if !identifier.MatchString(p.key) || p.key == "folders" {
		g.refuse(where, "%q cannot name a Stack (a lowercase RFC 1123 label, not folders)", p.key)
		return nil
	}
	sas := map[string]bool{}
	for _, q := range getMap(p.data, "service_accounts") {
		sas[q.Key] = true
	}
	sc := &scope{project: p, sas: sas}
	u := &unit{name: p.key, source: p.rel}
	pid := g.projectID(p)

	keep := false
	switch dp := g.effective(p, "deletion_policy"); dp {
	case nil, "DELETE":
	case "ABANDON":
		keep = true
	case "PREVENT":
		g.refuse(where+": deletion_policy", "PREVENT: %s", g.charts[chartProject].Refused["deletion_policy"].Reason)
	default:
		g.refuse(where+": deletion_policy", "%v is not DELETE, ABANDON or PREVENT", dp)
	}

	// The project component: name, prefix, parent and billing first, then
	// the file's keys in its order, then what only defaults set.
	in := convert.Ordered{}
	set(&in, "name", g.projectName(p))
	if pre := g.projectPrefix(p); pre != nil {
		set(&in, "prefix", pre)
	}
	if v := val(p.data, "descriptive_name"); v != nil {
		set(&in, "descriptive_name", v)
	}
	for _, k := range []string{"parent", "billing_account"} {
		if v := g.effective(p, k); v != nil {
			set(&in, k, g.resolveAll(v, sc, where+": "+k))
		}
	}
	moves := map[string]*saMoves{}
	for _, q := range p.data {
		w := where + ": " + q.Key
		switch q.Key {
		case "name", "prefix", "descriptive_name", "parent", "billing_account", "deletion_policy", "service_accounts", "buckets":
		case "labels", "services", "contacts", "metric_scopes", "tag_bindings":
			if v := g.effective(p, q.Key); !isEmpty(v) {
				set(&in, q.Key, g.resolveAll(v, sc, w))
			}
		case "data_access_logs":
			set(&in, "logging_data_access", g.resolveAll(q.Value, sc, w))
		case "iam", "iam_by_principals", "iam_by_principals_additive", "iam_bindings", "iam_bindings_additive", "iam_by_principals_conditional":
			g.projectIAM(&in, q.Key, q.Value, sc, pid, moves, w)
		default:
			if r, ok := notMapped[q.Key]; ok {
				g.refuse(w, "%s", r)
				continue
			}
			set(&in, q.Key, g.resolveAll(q.Value, sc, w))
		}
	}
	for _, k := range computed {
		if _, ok := get(p.data, k); ok {
			continue
		}
		if v := g.effective(p, k); !isEmpty(v) {
			set(&in, k, g.resolveAll(v, sc, where+": "+k))
		}
	}
	u.components = append(u.components, &component{name: "project", chart: g.charts[chartProject], inputs: in, keep: keep})

	for _, q := range getMap(p.data, "service_accounts") {
		opts, _ := q.Value.(convert.Ordered)
		u.components = append(u.components, g.serviceAccount(p, q.Key, opts, sc, pid, moves[q.Key], where+": service_accounts."+q.Key))
	}
	for _, q := range getMap(p.data, "buckets") {
		opts, _ := q.Value.(convert.Ordered)
		c := g.bucket(p, q.Key, opts, sc, where+": buckets."+q.Key)
		c.keep = keep
		u.components = append(u.components, c)
	}
	for k := range moves {
		if !sas[k] {
			g.refuse(where, "IAM for service account %s, which the project does not make", k)
		}
	}
	return u
}

// saMoves are the project's grants to one of its own service accounts,
// moved to the account's side (decision 8: the account's component needs
// the project's ID, so the project's cannot also need the account's email).
type saMoves struct {
	roles    []any
	bindings convert.Ordered
}

func moveFor(m map[string]*saMoves, sa string) *saMoves {
	if m[sa] == nil {
		m[sa] = &saMoves{}
	}
	return m[sa]
}

// projectIAM maps one of the project's IAM variables.
func (g *gen) projectIAM(in *convert.Ordered, k string, v any, sc *scope, pid string, moves map[string]*saMoves, where string) {
	m, _ := v.(convert.Ordered)
	ownSA := func(s string) string {
		if r := ownSAExpr.FindStringSubmatch(s); r != nil {
			return r[1]
		}
		return ""
	}
	switch k {
	case "iam":
		out := convert.Ordered{}
		for _, q := range m {
			role, _ := g.resolve(q.Key, sc, where+"."+q.Key)
			var keep []any
			members, _ := g.resolveAll(q.Value, sc, where+"."+q.Key).([]any)
			for _, mem := range members {
				s, _ := mem.(string)
				if sa := ownSA(s); sa != "" {
					mv := moveFor(moves, sa)
					mv.roles = appendRole(mv.roles, role)
					continue
				}
				keep = append(keep, mem)
			}
			if len(keep) > 0 {
				out = append(out, convert.Pair{Key: role, Value: keep})
			}
		}
		if len(out) > 0 {
			set(in, "iam", out)
		}
	case "iam_by_principals", "iam_by_principals_additive":
		g.byPrincipals(in, k, m, sc, where, func(sa string, roles []any) {
			mv := moveFor(moves, sa)
			for _, r := range roles {
				mv.roles = appendRole(mv.roles, r)
			}
		})
	case "iam_bindings_additive":
		add := getMap(*in, "iam_bindings_additive")
		for _, q := range m {
			b, _ := g.resolveAll(q.Value, sc, where+"."+q.Key).(convert.Ordered)
			if sa := ownSA(fmt.Sprint(val(b, "member"))); sa != "" {
				mb := convert.Ordered{{Key: "project_id", Value: pid}, {Key: "role", Value: val(b, "role")}}
				if c := val(b, "condition"); c != nil {
					mb = append(mb, convert.Pair{Key: "condition", Value: c})
				}
				mv := moveFor(moves, sa)
				set(&mv.bindings, q.Key, mb)
				continue
			}
			set(&add, q.Key, b)
		}
		if len(add) > 0 {
			set(in, "iam_bindings_additive", add)
		}
	case "iam_bindings":
		b := g.resolveAll(m, sc, where).(convert.Ordered)
		for _, q := range b {
			members, _ := val(q.Value.(convert.Ordered), "members").([]any)
			for _, mem := range members {
				if sa := ownSA(fmt.Sprint(mem)); sa != "" {
					g.refuse(where+"."+q.Key, "names the project's own service account %s: its grant cannot move to the account's side (the binding is authoritative for its role), and left here the project would need the account's email while the account needs the project's ID, a cycle", sa)
				}
			}
		}
		set(in, k, b)
	case "iam_by_principals_conditional":
		for _, q := range m {
			principal, ok := g.resolve(q.Key, sc, where+"."+q.Key)
			if ok && strings.Contains(principal, "${") {
				g.refuse(where+"."+q.Key, "keyed by a principal that is an expression (%s): a map key is never evaluated, and conditional grants are not rewritten yet", principal)
			}
		}
		set(in, k, g.resolveAll(m, sc, where))
	}
}

// byPrincipals maps a map keyed by principal (iam_by_principals,
// iam_by_principals_additive). A principal that is one of the project's own
// service accounts goes to own when it is set; another that is an
// expression becomes iam_bindings_additive entries, the principal a value:
// a map key is never evaluated (templates.md A23).
func (g *gen) byPrincipals(in *convert.Ordered, k string, m convert.Ordered, sc *scope, where string, own func(sa string, roles []any)) {
	out := convert.Ordered{}
	for _, q := range m {
		principal, ok := g.resolve(q.Key, sc, where+"."+q.Key)
		if !ok {
			continue
		}
		roles, _ := g.resolveAll(q.Value, sc, where+"."+q.Key).([]any)
		if r := ownSAExpr.FindStringSubmatch(principal); r != nil && own != nil {
			own(r[1], roles)
			continue
		}
		if strings.Contains(principal, "${") {
			add := getMap(*in, "iam_bindings_additive")
			for _, r := range roles {
				set(&add, strings.TrimPrefix(q.Key, "$")+":"+fmt.Sprint(r), convert.Ordered{{Key: "member", Value: principal}, {Key: "role", Value: r}})
			}
			set(in, "iam_bindings_additive", add)
			continue
		}
		out = append(out, convert.Pair{Key: principal, Value: roles})
	}
	if len(out) > 0 {
		set(in, k, out)
	}
}

func appendRole(l []any, r any) []any {
	if containsAny(l, r) {
		return l
	}
	return append(l, r)
}

func (g *gen) serviceAccount(p *projectFile, name string, opts convert.Ordered, sc *scope, pid string, mv *saMoves, where string) *component {
	in := convert.Ordered{}
	set(&in, "project_id", "${components.project.outputs.project_id}")
	set(&in, "name", name)
	for _, k := range []string{"display_name", "description"} {
		if v := val(opts, k); v != nil {
			set(&in, k, g.resolveAll(v, sc, where+"."+k))
		}
	}
	// iam_project_roles: the file's, then iam_self_roles and what moved
	// from the project, on the project's ID.
	roles := convert.Ordered{}
	if r, ok := g.resolveAll(val(opts, "iam_project_roles"), sc, where+".iam_project_roles").(convert.Ordered); ok {
		roles = r
	}
	self, _ := val(roles, pid).([]any)
	if sr, ok := g.resolveAll(val(opts, "iam_self_roles"), sc, where+".iam_self_roles").([]any); ok {
		for _, r := range sr {
			self = appendRole(self, r)
		}
	}
	bindings := convert.Ordered{}
	if b, ok := g.resolveAll(val(opts, "iam_project_bindings"), sc, where+".iam_project_bindings").(convert.Ordered); ok {
		bindings = b
	}
	if mv != nil {
		for _, r := range mv.roles {
			self = appendRole(self, r)
		}
		for _, b := range mv.bindings {
			set(&bindings, b.Key, b.Value)
		}
	}
	if len(self) > 0 {
		set(&roles, pid, self)
	}
	if len(roles) > 0 {
		set(&in, "iam_project_roles", roles)
	}
	if len(bindings) > 0 {
		set(&in, "iam_project_bindings", bindings)
	}
	for _, q := range opts {
		w := where + "." + q.Key
		switch {
		case q.Key == "display_name" || q.Key == "description" || q.Key == "iam_self_roles" || q.Key == "iam_project_roles" || q.Key == "iam_project_bindings":
		case saKeys[q.Key]:
			set(&in, q.Key, g.resolveAll(q.Value, sc, w))
		default:
			g.refuse(w, "fabric's factory does not read %s on a service account", q.Key)
		}
	}
	if !identifier.MatchString("sa-" + name) {
		g.refuse(where, "sa-%s cannot name a component (a lowercase RFC 1123 label)", name)
	}
	return &component{name: "sa-" + name, chart: g.charts[chartSA], inputs: in}
}

func (g *gen) bucket(p *projectFile, key string, opts convert.Ordered, sc *scope, where string) *component {
	in := convert.Ordered{}
	set(&in, "project_id", "${components.project.outputs.project_id}")
	if pre := first(val(g.override, "prefix"), val(opts, "prefix"), g.projectPrefix(p), val(g.defaults, "prefix")); pre != nil {
		set(&in, "prefix", pre)
	}
	name := val(opts, "name")
	if name == nil {
		name = g.projectName(p) + "-" + key
	}
	set(&in, "name", g.resolveAll(name, sc, where+".name"))
	if loc := first(val(getMap(g.override, "locations"), "storage"), val(opts, "location"), val(getMap(g.defaults, "locations"), "storage")); loc != nil {
		set(&in, "location", g.resolveAll(loc, sc, where+".location"))
	}
	if fd := first(val(getMap(g.override, "bucket"), "force_destroy"), val(opts, "force_destroy"), val(getMap(g.defaults, "bucket"), "force_destroy")); fd != nil {
		set(&in, "force_destroy", fd)
	}
	for _, q := range opts {
		switch q.Key {
		case "prefix", "name", "location", "force_destroy":
		case "create":
			set(&in, "bucket_create", q.Value)
		case "iam_by_principals":
			m, _ := q.Value.(convert.Ordered)
			g.byPrincipals(&in, q.Key, m, sc, where+"."+q.Key, nil)
		default:
			set(&in, q.Key, g.resolveAll(q.Value, sc, where+"."+q.Key))
		}
	}
	if !identifier.MatchString("bucket-" + key) {
		g.refuse(where, "bucket-%s cannot name a component (a lowercase RFC 1123 label)", key)
	}
	return &component{name: "bucket-" + key, chart: g.charts[chartGCS], inputs: in}
}

// ---- checks ---------------------------------------------------------------

// checkUnit refuses what the composed charts refuse, a value a chart does
// not have, a map key that holds an expression, and an expression on a
// component that is missing, the component itself, or part of a cycle.
func (g *gen) checkUnit(u *unit) {
	byName := map[string]*component{}
	for _, c := range u.components {
		byName[c.name] = c
	}
	deps := map[string][]string{}
	for _, c := range u.components {
		where := u.source + ": " + c.name
		allowAlpha := val(c.inputs, "allow_alpha") == true
		for _, p := range c.inputs {
			w := where + ": " + p.Key
			if _, ok := c.chart.Values[p.Key]; !ok {
				g.refuse(w, "%s has no value %s", c.chart.Package, p.Key)
				continue
			}
			if r, ok := c.chart.Refused[p.Key]; ok {
				gaps, _ := convert.Refusals(r, plain(p.Value), w, allowAlpha)
				for _, gap := range gaps {
					g.refused = append(g.refused, Refusal{Where: where, Text: strings.TrimPrefix(gap, where+": ")})
				}
			}
			walkKeys(p.Value, w, func(k, at string) {
				if strings.Contains(k, "${") {
					g.refuse(at, "the key %q is an expression, and a map key is never evaluated (templates.md A23)", k)
				}
			})
		}
		for _, m := range exprRef.FindAllStringSubmatch(stringsOf(c.inputs), -1) {
			d := m[1]
			switch {
			case d == c.name:
				g.refuse(where, "an expression on %s itself: a value it knows only once it is made (fabric's second IAM pass)", c.name)
			case byName[d] == nil:
				g.refuse(where, "an expression on %s, which this Instance does not have", d)
			case !contains(deps[c.name], d):
				deps[c.name] = append(deps[c.name], d)
			}
		}
	}
	// Cycles: Bundt orders components by their expressions and blocks a cycle.
	state := map[string]int{}
	var visit func(n string, trail []string)
	visit = func(n string, trail []string) {
		switch state[n] {
		case 1:
			g.refuse(u.source, "a cycle of expressions: %s", strings.Join(append(trail, n), " -> "))
			return
		case 2:
			return
		}
		state[n] = 1
		for _, d := range deps[n] {
			visit(d, append(trail, n))
		}
		state[n] = 2
	}
	for _, c := range u.components {
		visit(c.name, nil)
	}
}

func walkKeys(v any, where string, fn func(k, at string)) {
	switch x := v.(type) {
	case convert.Ordered:
		for _, p := range x {
			fn(p.Key, where+"."+p.Key)
			walkKeys(p.Value, where+"."+p.Key, fn)
		}
	case []any:
		for i, e := range x {
			walkKeys(e, fmt.Sprintf("%s[%d]", where, i), fn)
		}
	}
}

func stringsOf(v any) string {
	var b strings.Builder
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case string:
			b.WriteString(x)
			b.WriteByte('\n')
		case convert.Ordered:
			for _, p := range x {
				walk(p.Value)
			}
		case []any:
			for _, e := range x {
				walk(e)
			}
		}
	}
	walk(v)
	return b.String()
}

// plain turns ordered maps into map[string]any, as the chart refusals expect.
func plain(v any) any {
	switch x := v.(type) {
	case convert.Ordered:
		m := map[string]any{}
		for _, p := range x {
			m[p.Key] = plain(p.Value)
		}
		return m
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = plain(e)
		}
		return out
	}
	return v
}
