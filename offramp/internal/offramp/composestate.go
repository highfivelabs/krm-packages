// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ComposeStateOptions steer ComposeState.
type ComposeStateOptions struct {
	// Out is where each chart is written (<Out>/<chart>/).
	Out string
	// Single composes the whole state into one chart with this name;
	// otherwise each root module call (and the root module) is its own.
	Single string
	// Calls limits which calls are composed (by component name); empty is all.
	Calls       []string
	Prefix      string // prepended to each chart's name
	Environment string
	Moniker     string
	Version     string
	StatePath   string
}

// Composed is one chart composed from a state.
type Composed struct {
	Spec   *ChartSpec
	Dir    string
	Verify *Verification
}

// ComposeState composes, compiles and verifies one chart per call.
func ComposeState(res *Result, st *State, opt ComposeStateOptions) ([]*Composed, error) {
	groups := map[string][]*Object{}
	var order []string
	for _, o := range res.Objects {
		name := componentName(o.Call)
		if opt.Single != "" {
			name = opt.Single
		}
		if _, ok := groups[name]; !ok {
			order = append(order, name)
		}
		groups[name] = append(groups[name], o)
	}
	want := map[string]bool{}
	for _, c := range opt.Calls {
		want[componentName(c)] = true
	}
	var out []*Composed
	for _, name := range order {
		if len(want) > 0 && !want[name] {
			continue
		}
		objs := groups[name]
		chart := sanitize(strings.Trim(opt.Prefix+"-"+name, "-"))
		call := ""
		if opt.Single == "" {
			call = objs[0].Call
		}
		spec, err := Compose(objs, ComposeOptions{
			Name: chart, Version: opt.Version, Environment: opt.Environment, Moniker: opt.Moniker,
			Source: SpecSource{Kind: "terraform-state", State: filepath.Base(opt.StatePath), Tool: st.Tool, Form: st.Form, TerraformVersion: st.TerraformVersion, Call: call},
		})
		if err != nil {
			return out, fmt.Errorf("%s: %w", name, err)
		}
		dir := filepath.Join(opt.Out, chart)
		if err := Compile(spec, dir); err != nil {
			return out, fmt.Errorf("%s: compile: %w", name, err)
		}
		v := Verify(spec, dir, objs)
		if err := os.WriteFile(filepath.Join(dir, "verify.md"), []byte(VerifyReport(v)), 0o644); err != nil {
			return out, err
		}
		out = append(out, &Composed{Spec: spec, Dir: dir, Verify: v})
	}
	return out, nil
}

// ComposeReport summarizes the composed charts.
func ComposeReport(cs []*Composed, res *Result) string {
	var b strings.Builder
	b.WriteString("# Charts composed from a Terraform state\n\n| Chart | Resources | Values | Diff | Collisions | Verified |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, c := range cs {
		ok := "yes"
		if !c.Verify.OK() {
			ok = "**no**"
		}
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d | %s |\n", c.Spec.Chart.Name, len(c.Spec.Resources), len(c.Spec.Values), len(c.Verify.Differences), len(c.Verify.Collisions), ok)
	}
	if len(res.Findings) > 0 {
		b.WriteString("\n## Not carried by the conversion\n\n")
		for _, f := range res.Findings {
			fmt.Fprintf(&b, "- `%s` (%s, %s): %s\n", f.Source, f.Type, f.Class, f.Reason)
		}
	}
	return b.String()
}
