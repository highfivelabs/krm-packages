# The factory generator

`catalogtool factory` (`task factory`) reads a Cloud Foundation Fabric
`project-factory` tree and writes the Bundt documents that replace it, as
`docs/design.md` ("Submodules and factories", "Factories, worked") designs
it: one Stack and one Instance for the folders, and one Stack and one
Instance per project file. Each Stack lists its components inline (the
escape hatch, Bundt's L2.8): `gcp-folder`, `gcp-project`,
`gcp-iam-service-account` and `gcp-gcs`, at the versions their charts
have here. Built 2026-11-15 (RECPF).

```sh
task factory -- -config ../my-factory/factory.yaml -out /tmp/docs -folder-ids ids.yaml
```

| Flag | Meaning |
| --- | --- |
| `-config` | the generator's config (below); paths in it are relative to its directory, the factory root |
| `-out` | an empty or absent directory: a file left from an earlier run would keep a deleted project in the plan |
| `-folder-ids` | the folders' IDs, `team-a/app-0: folders/3456789012`: the folders Instance's outputs once it is applied |
| `-folders-only` | the first pass: the folders' documents alone, before the folders exist |
| `-commit` | the factory repository's commit for `hfl.dev/factory` (default: `git rev-parse --short HEAD` there) |
| `-team`, `-environment` | the Team owning the Stacks (`gcp-factory`) and the Instances' Environment (`gcp-org`, so `gcp-factory/gcp-org`) |

## The config

```yaml
defaults:                  # fabric's data_defaults
  prefix: hfl
  deletion_policy: ABANDON
overrides: {}              # fabric's data_overrides
merges: {}                 # fabric's data_merges
context:                   # fabric's context variable, as is
  folder_ids: {default: organizations/1234567890}
  storage_buckets: {log-bucket: hfl-central-logs}
folders: data/folders      # default folders
projects: data/projects    # default projects
```

`defaults` takes `billing_account`, `bucket.force_destroy`, `contacts`,
`deletion_policy`, `labels`, `locations.storage`, `metric_scopes`,
`parent`, `prefix`, `services`, `tag_bindings`; `overrides` the same but
`labels` and `metric_scopes`; `merges` `contacts`, `labels`,
`metric_scopes`, `services`, `tag_bindings`. They apply as fabric applies
them (`projects-defaults.tf`, `projects-buckets.tf`). Anything else is
refused by name.

## What it writes

The mapping is the table in `docs/design.md`, "Factories, worked":

- A project file is a Stack and an Instance named for the file. The
  project's keys are the `project` component's inputs; `data_access_logs`
  is `logging_data_access`. Fabric's `prefix` applies to the project and
  its buckets; a bucket is named `<project name>-<key>` unless it sets
  `name`, as fabric names it.
- `service_accounts.<n>` is `sa-<n>`; `iam_self_roles` is
  `iam_project_roles` on the project's ID.
- `buckets.<n>` is `bucket-<n>`; `create` is `bucket_create`.
- A folder (`<path>/.config.yaml`) is a component `folder-<path with - for
  />` of the `folders` Instance; a nested folder's parent is an expression
  on its parent's component, a top folder's is `$folder_ids:default` from
  the context unless it sets `parent`. A project file inside the folders
  tree is a project whose parent is its directory's folder.
- Lookups: `$iam_principals:service_accounts/_self_/<sa>` (or this
  project's key) is `${components.sa-<sa>.outputs.iam_email}`;
  `service_agents/_self_/<agent>` is the agent's identity from fabric's
  `service-agents.yaml` (copied here unchanged, Apache-2.0, Google LLC)
  with `${components.project.outputs.number}`; another project's account
  is its email, a literal; `$project_ids:<project>` is the project's ID, a
  literal; `$folder_ids:<folder>` is the ID from `-folder-ids`; any other
  lookup is the literal from `context`.
- A project's grant to one of its own accounts (in `iam`,
  `iam_by_principals`, `iam_by_principals_additive` or
  `iam_bindings_additive`) moves to the account's `iam_project_roles` or
  `iam_project_bindings` (decision 8: the cycle otherwise). A principal key
  that is an expression becomes an `iam_bindings_additive` entry named
  `<lookup>:<role>`.
- `deletion_policy: ABANDON` (the file's, or the defaults') is
  `cloudResources: keep` on the Instance's `project` and bucket
  components; `DELETE` is Bundt's default.
- `hfl.dev/factory: <file>@<commit>` is on every Stack and Instance.

`testdata/worked` is the design's worked example as a golden test
(`go test ./factory`, in `task tools-test`, so in `task check`); a chart's
new version changes its output, and `task factory-update` accepts it.

## What it refuses

A refusal anywhere writes nothing (a Team-scoped plan deletes what it does
not hold, so a partial tree would delete projects). It refuses, each by
name:

- what the composed charts refuse (their `upstream.yaml` `refused`, as
  `task convert` applies it), and a key a chart has no value for;
- what fabric's factory makes with a module the generator does not map
  yet: `automation`, `billing_budgets`, `datasets`, `kms`, `log_buckets`,
  `pubsub_topics`, `project_template`, a project's `factories_config`, and
  `dns_threat_detector` (no stable kind, and the project chart has no
  value for it);
- a lookup it cannot resolve: a folder of this factory with no ID yet,
  another project's number or service agent (cross-Instance references
  are a gap planned in Bundt), a key the context lacks, a value holding
  `${` (fabric's condition templating);
- what would make a cycle: an authoritative `iam_bindings` entry or a
  conditional grant naming the project's own account, the project's own
  service agent in the project's IAM (an expression on itself), accounts
  granting each other.

## Not yet

- `net-vpc-factory` is next: the design names it a composite factory but
  works only the project factory; its files are `net-vpc` and
  `net-vpc-firewall` values per file.
- Budgets, project templates, `automation`, KMS, Pub/Sub, datasets and log
  buckets: each a component of a chart the catalog has, mapped the same
  way when a factory needs it.
- Folder IDs and other projects' numbers as literals go away when Bundt
  can bind one Instance's outputs in another (contracts, exports and
  bindings).
