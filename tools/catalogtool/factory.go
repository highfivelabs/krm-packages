// Copyright 2026 High Five Labs
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/highfivelabs/catalog/tools/convert"
	"github.com/highfivelabs/catalog/tools/factory"
)

// factoryCmd: catalogtool factory -config FILE -out DIR [flags]
// (tools/factory/README.md).
func factoryCmd(args []string) error {
	fs := flag.NewFlagSet("factory", flag.ContinueOnError)
	config := fs.String("config", "", "the generator's config file (defaults, overrides, merges, context, folders, projects); paths in it are relative to its directory, the factory root")
	out := fs.String("out", "", "an empty (or absent) directory to write the documents into")
	ids := fs.String("folder-ids", "", "the folders' IDs, a YAML map of a folder's path in the tree to folders/N: the folders Instance's outputs")
	commit := fs.String("commit", "", "the factory repository's commit, for hfl.dev/factory (default: git rev-parse --short HEAD in the factory root)")
	team := fs.String("team", "gcp-factory", "the Team that owns the Stacks")
	env := fs.String("environment", "gcp-org", "the Environment of the Instances (of that Team)")
	foldersOnly := fs.Bool("folders-only", false, "write the folders' documents alone (the first pass, before the folders exist)")
	catalog := fs.String("catalog", "", "the catalog repository's root (default: found above the working directory)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *config == "" || *out == "" || fs.NArg() > 0 {
		return fmt.Errorf("usage: catalogtool factory -config FILE -out DIR [-folder-ids FILE] [-commit SHA] [-team T] [-environment E] [-folders-only] [-catalog DIR]")
	}
	root := *catalog
	if root == "" {
		r, err := repoRoot()
		if err != nil {
			return err
		}
		root = r
	}
	repo, err := convert.OpenRepo(root)
	if err != nil {
		return err
	}
	o := factory.Options{Repo: repo, Config: *config, Commit: *commit, Team: *team, Environment: *env, FoldersOnly: *foldersOnly}
	if *ids != "" {
		if o.FolderIDs, err = factory.ReadFolderIDs(*ids); err != nil {
			return err
		}
	}
	if o.Commit == "" {
		b, err := exec.Command("git", "-C", filepath.Dir(*config), "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			return fmt.Errorf("no -commit, and git rev-parse in %s failed: %v", filepath.Dir(*config), err)
		}
		o.Commit = strings.TrimSpace(string(b))
	}
	files, err := factory.Generate(o)
	if err != nil {
		return err
	}
	if err := factory.Write(*out, files); err != nil {
		return err
	}
	fmt.Printf("factory: wrote %d files to %s\n", len(files), *out)
	return nil
}
