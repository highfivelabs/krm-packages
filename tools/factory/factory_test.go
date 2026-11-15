// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/highfivelabs/catalog/tools/convert"
)

func repo(t *testing.T) *convert.Repo {
	t.Helper()
	r, err := convert.OpenRepo(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func worked(t *testing.T, foldersOnly bool) []File {
	t.Helper()
	ids, err := ReadFolderIDs(filepath.Join("testdata", "worked", "folder-ids.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := Generate(Options{
		Repo:        repo(t),
		Config:      filepath.Join("testdata", "worked", "factory.yaml"),
		FolderIDs:   ids,
		Commit:      "4f2a9c1",
		FoldersOnly: foldersOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestWorked is the worked example of docs/design.md ("Factories,
// worked") as a golden test: fabric's file in, the documents the design
// shows out. UPDATE=1 rewrites testdata/worked/out.
func TestWorked(t *testing.T) {
	files := worked(t, false)
	out := filepath.Join("testdata", "worked", "out")
	if os.Getenv("UPDATE") != "" {
		if err := os.RemoveAll(out); err != nil {
			t.Fatal(err)
		}
		if err := Write(out, files); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(files) {
		t.Errorf("%s holds %d files, the generator wrote %d (UPDATE=1 to accept)", out, len(entries), len(files))
	}
	for _, f := range files {
		want, err := os.ReadFile(filepath.Join(out, f.Name))
		if err != nil {
			t.Fatalf("%v (UPDATE=1 to accept)", err)
		}
		if string(want) != f.Content {
			t.Errorf("%s differs (UPDATE=1 to accept; a chart's new version changes it too).\nwant:\n%s\ngot:\n%s", f.Name, want, f.Content)
		}
	}
	// What the design's table says, checked apart from the golden text.
	var project string
	for _, f := range files {
		if f.Name == "dev-ta-app0-be.yaml" {
			project = f.Content
		}
	}
	for _, s := range []string{
		`parent: folders/3456789012`,
		`"${components.sa-app-0-fe.outputs.iam_email}"`,
		`"serviceAccount:service-${components.project.outputs.number}@compute-system.iam.gserviceaccount.com"`,
		`log_bucket: hfl-central-logs`,
		`hfl-dev-ta-app0-be:`,
		`hfl.dev/factory: "data/projects/dev-ta-app0-be.yaml@4f2a9c1"`,
		`cloudResources: keep`,
	} {
		if !strings.Contains(project, s) {
			t.Errorf("dev-ta-app0-be.yaml lacks %s", s)
		}
	}
	if strings.Contains(project, "iam_by_principals") {
		t.Errorf("the project's grant to its own account stayed on the project:\n%s", project)
	}
}

func TestFoldersOnly(t *testing.T) {
	files := worked(t, true)
	if len(files) != 1 || files[0].Name != "folders.yaml" {
		t.Fatalf("want folders.yaml alone, got %v", files)
	}
	for _, s := range []string{`parent: organizations/1234567890`, `parent: "${components.folder-team-a.outputs.id}"`} {
		if !strings.Contains(files[0].Content, s) {
			t.Errorf("folders.yaml lacks %s:\n%s", s, files[0].Content)
		}
	}
}

// tree writes a factory under a temporary directory: files maps a path
// (relative to the factory root) to its content.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, c := range files {
		f := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "factory.yaml")
}

const config = "context:\n  folder_ids: {default: organizations/1}\n"

func TestRefusals(t *testing.T) {
	cases := []struct {
		name    string
		project string
		config  string
		ids     map[string]string
		want    []string
	}{
		{
			name:    "a module the generator does not map",
			project: "kms:\n  keyrings: {k: {location: eu}}\npubsub_topics: {t: {}}\n",
			want:    []string{"kms: KMS key rings", "pubsub_topics: Pub/Sub topics"},
		},
		{
			name:    "the DNS threat detector",
			project: "dns_threat_detector: {enabled: true}\n",
			want:    []string{"dns_threat_detector: google_network_security_dns_threat_detector has no stable kind"},
		},
		{
			name:    "a value the project chart refuses",
			project: "default_service_account: delete\ncompute_metadata: {a: b}\n",
			want:    []string{`"delete"`, "compute_metadata: refused by the chart (gap)"},
		},
		{
			name:    "PREVENT",
			project: "deletion_policy: PREVENT\n",
			want:    []string{"deletion_policy: PREVENT: DELETE and ABANDON only"},
		},
		{
			name:    "a factory folder with no ID yet",
			project: "parent: $folder_ids:team-x\n",
			config:  config + "folders: folders\n",
			want:    []string{"the folder team-x is made by the folders Instance and has no ID here"},
		},
		{
			name:    "a lookup the context lacks",
			project: "iam:\n  roles/viewer: [$iam_principals:nobody]\n",
			want:    []string{"$iam_principals:nobody: not made by this factory and not in the factory's context"},
		},
		{
			name:    "the project's own agent on the project",
			project: "services: [compute.googleapis.com]\niam:\n  roles/editor: [$iam_principals:service_agents/_self_/compute]\n",
			want:    []string{"an expression on project itself"},
		},
		{
			name: "an authoritative binding naming its own account",
			project: "service_accounts: {a: {}}\niam_bindings:\n  b:\n    role: roles/viewer\n" +
				"    members: [$iam_principals:service_accounts/_self_/a]\n",
			want: []string{"names the project's own service account a"},
		},
		{
			name: "two accounts granting each other",
			project: "service_accounts:\n  a: {iam: {roles/iam.serviceAccountUser: [$iam_principals:service_accounts/_self_/b]}}\n" +
				"  b: {iam: {roles/iam.serviceAccountUser: [$iam_principals:service_accounts/_self_/a]}}\n",
			want: []string{"a cycle of expressions"},
		},
		{
			name:    "a key nobody reads",
			project: "frobnicate: true\n",
			want:    []string{"gcp-project has no value frobnicate"},
		},
		{
			name:    "fabric's condition templating",
			project: "iam_bindings_additive:\n  c: {role: roles/viewer, member: group:a@example.org, condition: {title: t, expression: \"${folder_ids.x}\"}}\n",
			want:    []string{"holds ${...}"},
		},
		{
			name:    "a config key it does not read",
			project: "name: p\n",
			config:  config + "data_overrides: {}\n",
			want:    []string{"data_overrides is not read by the generator"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := c.config
			if cfg == "" {
				cfg = config
			}
			files := map[string]string{"factory.yaml": cfg, "projects/p.yaml": c.project}
			if strings.Contains(cfg, "folders: folders") {
				files["folders/team-x/.config.yaml"] = "name: Team X\n"
			}
			_, err := Generate(Options{Repo: repo(t), Config: tree(t, files), Commit: "c0ffee0", FolderIDs: c.ids})
			var e *Error
			if !errors.As(err, &e) {
				t.Fatalf("want refusals, got %v", err)
			}
			for _, w := range c.want {
				if !strings.Contains(e.Error(), w) {
					t.Errorf("refusals lack %q:\n%s", w, e.Error())
				}
			}
		})
	}
}

// A project whose principal key is another expression (a service agent):
// the entry becomes an additive binding; another project's account is a
// literal email; $project_ids is the other project's ID.
func TestRewrites(t *testing.T) {
	cfg := tree(t, map[string]string{
		"factory.yaml": config + "defaults: {prefix: x}\n",
		"projects/a.yaml": "services: [compute.googleapis.com]\nservice_accounts: {s: {}}\nbuckets:\n  b:\n    iam_by_principals:\n" +
			"      $iam_principals:service_agents/_self_/compute: [roles/storage.objectViewer]\n",
		"projects/b.yaml": "iam:\n  roles/viewer: [$iam_principals:service_accounts/a/s]\n" +
			"service_accounts:\n  t:\n    iam_project_roles:\n      $project_ids:a: [roles/viewer]\n",
	})
	files, err := Generate(Options{Repo: repo(t), Config: cfg, Commit: "c0ffee0"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range files {
		got[f.Name] = f.Content
	}
	for name, want := range map[string][]string{
		"a.yaml": {"iam_bindings_additive:", `member: "serviceAccount:service-${components.project.outputs.number}@compute-system.iam.gserviceaccount.com"`, "name: a-b"},
		"b.yaml": {"serviceAccount:s@x-a.iam.gserviceaccount.com", "x-a:", "scope: gcp-factory/gcp-org"},
	} {
		for _, w := range want {
			if !strings.Contains(got[name], w) {
				t.Errorf("%s lacks %q:\n%s", name, w, got[name])
			}
		}
	}
	if strings.Contains(got["a.yaml"], "cloudResources") {
		t.Errorf("no deletion_policy, yet keep:\n%s", got["a.yaml"])
	}
}

func TestWriteRefusesAFullDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.yaml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, []File{{Name: "a.yaml", Content: "x"}}); err == nil {
		t.Fatal("wrote into a directory holding files")
	}
}
