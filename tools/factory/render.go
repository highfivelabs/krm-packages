// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"bytes"
	_ "embed"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/highfivelabs/catalog/tools/convert"
)

// service-agents.yaml is fabric's list of service agents
// (modules/project/service-agents.yaml at the pinned commit, Apache-2.0,
// Google LLC), unchanged: the generator reads an agent's identity from it,
// as fabric's project module does, to write
// $iam_principals:service_agents/_self_/<agent>.
//
//go:embed service-agents.yaml
var serviceAgentsYAML []byte

type serviceAgent struct {
	Name     string   `yaml:"name"`
	Identity string   `yaml:"identity"`
	Aliases  []string `yaml:"aliases"`
}

func loadServiceAgents() ([]serviceAgent, error) {
	var a []serviceAgent
	err := yaml.Unmarshal(serviceAgentsYAML, &a)
	return a, err
}

func str(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

func quoted(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s, Style: yaml.DoubleQuotedStyle}
}

func mapping(kv ...*yaml.Node) *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: kv}
}

func node(v any) *yaml.Node {
	switch x := v.(type) {
	case convert.Ordered:
		n := mapping()
		for _, p := range x {
			n.Content = append(n.Content, keyNode(p.Key), node(p.Value))
		}
		return n
	case []any:
		n := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, e := range x {
			n.Content = append(n.Content, node(e))
		}
		return n
	case string:
		n := &yaml.Node{}
		if err := n.Encode(x); err != nil {
			panic(err)
		}
		if strings.Contains(x, "${") {
			n.Style = yaml.DoubleQuotedStyle
		}
		return n
	}
	n := &yaml.Node{}
	if err := n.Encode(v); err != nil {
		panic(err)
	}
	return n
}

func keyNode(k string) *yaml.Node {
	n := &yaml.Node{}
	if err := n.Encode(k); err != nil {
		panic(err)
	}
	return n
}

// render writes one unit's Stack (its components inline) and Instance.
func (g *gen) render(u *unit) string {
	ann := mapping(str("hfl.dev/factory"), quoted(u.source+"@"+g.o.Commit))
	var comps []*yaml.Node
	for _, c := range u.components {
		cn := mapping(
			str("name"), str(c.name),
			str("package"), mapping(str("name"), str(c.chart.Package), str("version"), quoted(c.chart.Version)),
		)
		if len(c.inputs) > 0 {
			cn.Content = append(cn.Content, str("inputs"), node(c.inputs))
		}
		comps = append(comps, cn)
	}
	stack := mapping(
		str("apiVersion"), str("hfl.dev/v1"),
		str("kind"), str("Stack"),
		str("metadata"), mapping(str("name"), str(u.name), str("scope"), str(g.o.Team), str("annotations"), ann),
		str("spec"), mapping(str("components"), &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Content: comps}),
	)
	spec := mapping(str("stack"), str(u.name))
	keep := mapping()
	for _, c := range u.components {
		if c.keep {
			keep.Content = append(keep.Content, str(c.name), mapping(str("cloudResources"), str("keep")))
		}
	}
	if len(keep.Content) > 0 {
		spec.Content = append(spec.Content, str("components"), keep)
	}
	instance := mapping(
		str("apiVersion"), str("hfl.dev/v1"),
		str("kind"), str("Instance"),
		str("metadata"), mapping(str("name"), str(u.name), str("scope"), str(g.o.Team+"/"+g.o.Environment), str("annotations"), ann),
		str("spec"), spec,
	)
	var b bytes.Buffer
	b.WriteString("# Written by the factory generator (catalogtool factory) from " + u.source + "\n# at " + g.o.Commit + ". Do not edit: edit the factory's files and run it again.\n")
	for i, d := range []*yaml.Node{stack, instance} {
		if i > 0 {
			b.WriteString("---\n")
		}
		enc := yaml.NewEncoder(&b)
		enc.SetIndent(2)
		if err := enc.Encode(d); err != nil {
			panic(err)
		}
		enc.Close()
	}
	return b.String()
}
