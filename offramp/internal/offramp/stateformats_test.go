// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package offramp

import (
	"strings"
	"testing"
)

const sf = "testdata/stateformats/"

// Every form of the gke state, from either tool, converts to the same
// objects as the raw Terraform state.
func TestStateFormsConvertAlike(t *testing.T) {
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	base, err := ReadState("testdata/synthetic/gke/terraform.tfstate")
	if err != nil {
		t.Fatal(err)
	}
	want := ConvertState(base, k, Options{})
	cases := map[string]struct{ tool, form string }{
		sf + "gke.terraform-1.17.2.show.json":         {"terraform", "show-json"},
		sf + "gke.opentofu.tfstate":                   {"opentofu", "state"},
		sf + "gke.opentofu-1.10.7.show.json":          {"opentofu", "show-json"},
		sf + "gke.deposed.tfstate":                    {"terraform", "state"},
		sf + "gke.deposed.terraform-1.17.2.show.json": {"terraform", "show-json"},
	}
	for path, c := range cases {
		t.Run(path, func(t *testing.T) {
			st, err := ReadState(path)
			if err != nil {
				t.Fatal(err)
			}
			if st.Tool != c.tool || st.Form != c.form {
				t.Errorf("tool %q form %q, want %q %q", st.Tool, st.Form, c.tool, c.form)
			}
			got := ConvertState(st, k, Options{})
			if len(got.Objects) != len(want.Objects) {
				t.Fatalf("%d objects, want %d", len(got.Objects), len(want.Objects))
			}
			var a, b []map[string]any
			for i := range got.Objects {
				a = append(a, got.Objects[i].Obj)
				b = append(b, want.Objects[i].Obj)
				if got.Objects[i].Source != want.Objects[i].Source {
					t.Errorf("object %d from %s, want %s", i, got.Objects[i].Source, want.Objects[i].Source)
				}
			}
			if d := append(Compare(a, b), metaDiffs(a, b)...); len(d) > 0 {
				t.Errorf("differs from the raw state: %v", d)
			}
		})
	}
}

// A deposed instance is not converted, and the result says so.
func TestDeposedSkipped(t *testing.T) {
	k, err := LoadKCC("testdata/kcc")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{sf + "gke.deposed.tfstate", sf + "gke.deposed.terraform-1.17.2.show.json"} {
		st, err := ReadState(path)
		if err != nil {
			t.Fatal(err)
		}
		res := ConvertState(st, k, Options{})
		found := false
		for _, f := range res.Findings {
			if strings.Contains(f.Source, `pools["apps"] (deposed 0f1e2d3c)`) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no finding for the deposed instance", path)
		}
	}
}

// Sensitive attributes read the same from both tools' raw states.
func TestSensitiveBothTools(t *testing.T) {
	for _, path := range []string{sf + "sensitive.terraform-1.17.2.tfstate", sf + "sensitive.opentofu-1.10.7.tfstate"} {
		st, err := ReadState(path)
		if err != nil {
			t.Fatal(err)
		}
		in := st.Instances()
		if len(in) != 1 || !in[0].Sensitive["input"] {
			t.Fatalf("%s: input not marked sensitive: %+v", path, in)
		}
		// Only the sensitive leaf goes; the rest of the attribute stays.
		if a, b := dig(in[0].Attrs, "input", "value", "a"), dig(in[0].Attrs, "input", "value", "b"); a != nil || b != "plain" {
			t.Errorf("%s: input.a = %v (want removed), input.b = %v (want plain)", path, a, b)
		}
	}
	// show -json marks the same attribute.
	if p := sensitivePaths(map[string]any{"input": map[string]any{"a": true}, "plain": false}); len(p) != 1 {
		t.Errorf("sensitivePaths = %v", p)
	}
}

// What cannot be read is refused with what to run instead.
func TestStateFormsRefused(t *testing.T) {
	for path, want := range map[string]string{
		sf + "opentofu-1.10.7.encrypted.tfstate": "tofu state pull",
		sf + "v3.tfstate":                        "Terraform 0.11 or earlier",
	} {
		if _, err := ReadState(path); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %v, want one naming %q", path, err, want)
		}
	}
	for doc, want := range map[string]string{
		`{"version": 5, "resources": []}`:         "newer than the converter knows",
		`{"format_version": "2.0", "values": {}}`: "reads 1.x",
		`{"format_version": "1.2"}`:               "no values",
		`{"serial": 1}`:                           "not a state",
	} {
		if _, err := ParseState([]byte(doc)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: error %v, want one naming %q", doc, err, want)
		}
	}
}

// A chart composed from OpenTofu's show -json records its source and
// verifies like one from Terraform's raw state.
func TestComposeFromOpenTofu(t *testing.T) {
	needHelm(t)
	for _, c := range composeFixture(t, sf+"gke.opentofu-1.10.7.show.json") {
		if !c.Verify.OK() {
			t.Errorf("%s not proven:\n%s", c.Spec.Chart.Name, VerifyReport(c.Verify))
		}
		if c.Spec.Source.Tool != "opentofu" || c.Spec.Source.Form != "show-json" {
			t.Errorf("%s: source %+v", c.Spec.Chart.Name, c.Spec.Source)
		}
	}
}
