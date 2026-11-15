// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// Write writes the files into dir, which must be empty or absent: a file
// left from an earlier run would stay in the plan after its project left
// the tree, and a Team-scoped plan deletes only what is absent.
func Write(dir string, files []File) error {
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s is not empty: write into an empty directory, so a project that left the tree leaves the plan", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), []byte(f.Content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// ReadFolderIDs reads the folders' IDs: a YAML map of a folder's path in
// the tree (team-a/app-0) to its ID (folders/N), the folders Instance's
// outputs.
func ReadFolderIDs(file string) (map[string]string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	ids := map[string]string{}
	if err := yaml.Unmarshal(data, &ids); err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return ids, nil
}
