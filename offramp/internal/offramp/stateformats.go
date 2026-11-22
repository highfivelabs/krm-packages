// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

// The state forms the converter reads, from Terraform and OpenTofu alike:
//
//   - A raw state snapshot, format version 4: what both write from
//     Terraform 0.12 on and in every OpenTofu release (a local
//     terraform.tfstate, or `terraform state pull` / `tofu state pull`).
//     The raw format carries no compatibility promise, so unknown fields
//     are ignored and only what the converter needs is read.
//   - `terraform show -json` / `tofu show -json` output (format_version
//     1.x): the stable, documented representation. Read it when the raw
//     form is not at hand, or to read an encrypted OpenTofu state, which
//     show -json decrypts with the root's configured keys.
//
// Refused, each with what to run instead: an OpenTofu-encrypted state
// (1.7+), format 3 or older (Terraform 0.11 and earlier), a format newer
// than 4, and a show -json major version other than 1.

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ParseState reads any supported state form into a State.
func ParseState(b []byte) (*State, error) {
	var probe struct {
		Version           *int            `json:"version"`
		FormatVersion     string          `json:"format_version"`
		EncryptionVersion string          `json:"encryption_version"`
		EncryptedData     json.RawMessage `json:"encrypted_data"`
		Values            json.RawMessage `json:"values"`
	}
	if err := json.Unmarshal(b, &probe); err != nil {
		return nil, fmt.Errorf("not a state: %w", err)
	}
	switch {
	case probe.EncryptionVersion != "" || len(probe.EncryptedData) > 0:
		return nil, fmt.Errorf("an OpenTofu-encrypted state (encryption_version %q): decrypt it with the root module's own keys, `tofu state pull > plain.tfstate` or `tofu show -json > state.json` in that root, and pass that file; the converter never holds state keys", probe.EncryptionVersion)
	case probe.FormatVersion != "":
		return parseShowJSON(b, probe.FormatVersion)
	case probe.Version == nil:
		return nil, fmt.Errorf("not a state: no version, format_version or encryption_version")
	case *probe.Version < 4:
		return nil, fmt.Errorf("state format version %d is Terraform 0.11 or earlier: read it with Terraform 0.12 or later (`terraform show -json > state.json`, which upgrades it in memory) and pass that file", *probe.Version)
	case *probe.Version > 4:
		return nil, fmt.Errorf("state format version %d is newer than the converter knows (4): pass `terraform show -json` or `tofu show -json` output instead, whose format is versioned for tools", *probe.Version)
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	s.Form = "state"
	s.Tool = toolOf(s.Resources)
	return &s, nil
}

// showJSON is the show -json state representation.
type showJSON struct {
	FormatVersion    string `json:"format_version"`
	TerraformVersion string `json:"terraform_version"`
	Values           *struct {
		RootModule showModule `json:"root_module"`
	} `json:"values"`
}

type showModule struct {
	Address      string         `json:"address"`
	Resources    []showResource `json:"resources"`
	ChildModules []showModule   `json:"child_modules"`
}

type showResource struct {
	Address         string         `json:"address"`
	Mode            string         `json:"mode"`
	Type            string         `json:"type"`
	Name            string         `json:"name"`
	Index           any            `json:"index"`
	ProviderName    string         `json:"provider_name"`
	Values          map[string]any `json:"values"`
	SensitiveValues map[string]any `json:"sensitive_values"`
	DependsOn       []string       `json:"depends_on"`
	DeposedKey      string         `json:"deposed_key"`
}

func parseShowJSON(b []byte, fv string) (*State, error) {
	if major := strings.SplitN(fv, ".", 2)[0]; major != "1" {
		return nil, fmt.Errorf("show -json format_version %s: the converter reads 1.x", fv)
	}
	var sj showJSON
	if err := json.Unmarshal(b, &sj); err != nil {
		return nil, err
	}
	if sj.Values == nil {
		return nil, fmt.Errorf("show -json output with no values: a plan, or an empty state? pass the state's (`show -json` with no plan file)")
	}
	s := &State{Version: 4, TerraformVersion: sj.TerraformVersion, Form: "show-json"}
	idx := map[string]int{} // module|mode|type|name|provider -> Resources index
	var walk func(m showModule)
	walk = func(m showModule) {
		for _, r := range m.Resources {
			prov := `provider["` + r.ProviderName + `"]`
			key := strings.Join([]string{m.Address, r.Mode, r.Type, r.Name, prov}, "|")
			i, ok := idx[key]
			if !ok {
				i = len(s.Resources)
				idx[key] = i
				s.Resources = append(s.Resources, StateResource{Module: m.Address, Mode: r.Mode, Type: r.Type, Name: r.Name, Provider: prov})
			}
			s.Resources[i].Instances = append(s.Resources[i].Instances, StateInstance{
				IndexKey:            r.Index,
				Attributes:          r.Values,
				Dependencies:        r.DependsOn,
				SensitiveAttributes: sensitivePaths(r.SensitiveValues),
				Deposed:             r.DeposedKey,
			})
		}
		for _, c := range m.ChildModules {
			walk(c)
		}
	}
	walk(sj.Values.RootModule)
	s.Tool = toolOf(s.Resources)
	return s, nil
}

// sensitivePaths turns show -json's sensitive_values (a mirror of the
// values with true where sensitive) into the raw state's path form: one
// path of get_attr and index steps per sensitive value.
func sensitivePaths(sv map[string]any) []any {
	var out []any
	var walk func(v any, path []any)
	walk = func(v any, path []any) {
		switch t := v.(type) {
		case bool:
			if t {
				out = append(out, append([]any(nil), path...))
			}
		case map[string]any:
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(t[k], append(path, map[string]any{"type": "get_attr", "value": k}))
			}
		case []any:
			for i, x := range t {
				walk(x, append(path, map[string]any{"type": "index", "value": map[string]any{"value": float64(i), "type": "number"}}))
			}
		}
	}
	walk(map[string]any(sv), nil)
	return out
}

// redact copies attrs with each sensitive path's value removed, and names
// the top-level attributes where a removed value was set.
func redact(attrs map[string]any, paths []any) (map[string]any, map[string]bool) {
	sens := map[string]bool{}
	if len(paths) == 0 {
		return attrs, sens
	}
	out := deepCopy(attrs).(map[string]any)
	for _, p := range paths {
		steps, ok := p.([]any)
		if !ok || len(steps) == 0 {
			continue
		}
		top, _ := stepKey(steps[0]).(string)
		if top == "" {
			continue
		}
		if removePath(out, steps) {
			sens[top] = true
		}
	}
	return out, sens
}

// stepKey is a path step's key: a string for get_attr (or a string map
// key), an int for a list index.
func stepKey(step any) any {
	m, ok := step.(map[string]any)
	if !ok {
		return nil
	}
	v := m["value"]
	if inner, ok := v.(map[string]any); ok { // {"value": 0, "type": "number"}
		v = inner["value"]
	}
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return v
}

// removePath nulls the value at a path; true when it held a value.
func removePath(v any, steps []any) bool {
	k := stepKey(steps[0])
	last := len(steps) == 1
	switch t := v.(type) {
	case map[string]any:
		ks, ok := k.(string)
		if !ok {
			return false
		}
		x, present := t[ks]
		if !present {
			// A dynamically typed value is stored as {"value": ..., "type": ...}
			// by both tools; the path goes through "value".
			if inner, wrapped := t["value"]; wrapped && len(t) == 2 && t["type"] != nil {
				return removePath(inner, steps)
			}
		}
		if !present {
			return false
		}
		if last {
			t[ks] = nil
			return !empty(x)
		}
		return removePath(x, steps[1:])
	case []any:
		i, ok := k.(int)
		if !ok || i < 0 || i >= len(t) {
			return false
		}
		if last {
			x := t[i]
			t[i] = nil
			return !empty(x)
		}
		return removePath(t[i], steps[1:])
	}
	return false
}

// toolOf names the tool from the providers' registry hosts.
func toolOf(rs []StateResource) string {
	tool := ""
	for _, r := range rs {
		switch {
		case strings.Contains(r.Provider, "registry.opentofu.org/"):
			return "opentofu"
		case strings.Contains(r.Provider, "registry.terraform.io/"):
			tool = "terraform"
		}
	}
	return tool
}

// DeposedAddresses lists the deposed instances the converter skips.
func (s *State) DeposedAddresses() []string {
	var out []string
	for _, r := range s.Resources {
		for _, in := range r.Instances {
			if in.Deposed == "" {
				continue
			}
			base := r.Type + "." + r.Name
			if r.Mode == "data" {
				base = "data." + base
			}
			if r.Module != "" {
				base = r.Module + "." + base
			}
			out = append(out, base+indexSuffix(in.IndexKey)+" (deposed "+in.Deposed+")")
		}
	}
	return out
}
