# The Terraform offramp

A prototype of a no-op, day-one import from Terraform into Bundt on Google
Cloud. It starts from a Terraform state, or from a scan of a folder or
organization. It produces Config Connector objects that adopt each resource
as it is, and the Bundt documents that hold them. Where a module call's
module has a catalog chart, the call becomes an Instance of that chart, with
the call's arguments as its values unchanged. It does so only after the
chart's render is checked against the state.

The design, the live results and the questions for the builder are in Bundt,
`drafts/tf-offramp/README.md` (branch `tf-offramp`).

| Path | What |
| --- | --- |
| `fixtures/` | four Terraform root modules built only from published modules pinned to exact releases (below); they may move to the `terraform-offramp-demo` repository, so every task takes `FIXTURES` |
| `fixtures-terragrunt/` | a Terragrunt estate: units whose source is an open-source module, and units whose source is the team's own wrapper module (below) |
| `cmd/offramp`, `internal/offramp` | the converter, the graduation to charts, `offramp compose` (charts authored from a state), `offramp synth`, the Terragrunt reader |
| `schema/` | the ChartSpec schema `offramp compose` writes (`docs/compose.md`) |
| `hack/scan.sh` | the scan: Cloud Asset Inventory, else the services' own list calls |
| `hack/live.sh` | the live proof of one shape |
| `hack/validate.sh` | `terraform fmt`, `init` and `validate` of every fixture; keeps each fixture's `modules.json` |
| `internal/offramp/testdata/` | Config Connector's CRDs and service mappings for the kinds the fixtures use; `modules/` (each fixture's `modules.json`), `live/` (states from live runs), `synthetic/` (states from `offramp synth`) |

## The fixtures: module calls and their charts

Every resource comes from a module call. Calls use terraform-google-modules,
the family in use. They use Cloud Foundation Fabric where the
catalog's existing chart ports Fabric (`net-vpc`, `project`,
`iam-service-account`, `gcs`), so every call has a chart that ports the same
module at the same release.

| Shape | Call | Module and release | Catalog chart | Status |
| --- | --- | --- | --- | --- |
| shared-vpc | `vpc` | fabric `net-vpc` v60.0.0 | `net-vpc` | exists (0.3.1 renders flow log defaults) |
| shared-vpc | `firewall_rules` | tgm `network//modules/firewall-rules` 18.3.1 | `firewall-rules` | built here |
| shared-vpc | `network_firewall_policy` | tgm `network//modules/network-firewall-policy` 18.3.1 | `network-firewall-policy` | built here (rules are a v1alpha1 kind: `allow_alpha`) |
| folders | `root`, `environments`, `teams` | tgm `folders` 5.2.0 | `folders` | built here |
| folders | `auditor` | tgm `iam//modules/folders_iam` 8.3.0 | `folders-iam` | built here |
| folders | `skip_default_network`, `resource_locations`, `require_os_login` | tgm `org-policy//modules/org_policy_v2` 7.4.0 | `org-policy-v2` | built here |
| folders | `hierarchical_firewall` | tgm `network//modules/hierarchical-firewall-policy` 18.3.1 | `hierarchical-firewall-policy` | built here |
| platform-project | `project` | fabric `project` v60.0.0 | `project` | exists; the render lacks fabric's default service agents and their grants (not ported), so the call stays a frozen Package |
| platform-project | `platform_deployer`, `platform_ci`, `checkout` | fabric `iam-service-account` v60.0.0 | `iam-service-account` | exists |
| platform-project | `checkout_assets` | fabric `gcs` v60.0.0 | `gcs` | exists |
| platform-project | `checkout_secrets` | fabric `secret-manager` v60.0.0 | `secret-manager` | exists (global secrets, no versions) |
| gke | `network` | fabric `net-vpc` v60.0.0 | `net-vpc` | exists |
| gke | `gke_standard` (`gke_autopilot` with `autopilot = true`) | tgm `kubernetes-engine//modules/beta-private-cluster` (`beta-autopilot-private-cluster`) 47.0.0 | `gke-cluster` (PR #16; `mode: autopilot`) | exists on its branch; its render sets `datapathProvider` and `initialNodeCount` where the planned state holds neither, so the call stays frozen until a live run settles it |

shared-vpc (on the old organization) and folders (on the new one) have
run live end to end. Since 2026-11-15 the folders and platform-project
shapes run against the new organization's live-tests folder (`OFFRAMP_*`
in `.env`, see `.env.example`):

- folders: **proved a no-op** (run `ofr-11160002-a4c3`, 2026-11-16, with
  the hierarchical firewall policy and without the organization
  policies): 14 resources applied, `terraform plan` empty, export 6 of 6,
  all five calls graduated to their charts with matching renders, the 14
  objects applied to kind with config-connector-extensions' CRDs and
  manager beside Config Connector's (its `IAMBinding` renders the
  authoritative bindings), preview 13 good and 0 bad with no `GCP_WRITE`
  (preview runs Config Connector's kinds only), 14 of 14 `Ready`
  `UpToDate`, no write in either manager's log, Bundt's server-side dry
  run and `terraform plan` both empty, and no Admin Activity entry by the
  run's account in the adoption window in the organization's, the
  live-tests folder's or the project's log (the run's own folders' logs
  were not readable from the VM, and went with the folders). Google Cloud grants a folder's
  creator `folderAdmin` and `folderEditor` on it, so the new
  organization's domain-restricted sharing (`iam.allowedPolicyMemberDomains`)
  sometimes refuses the create itself for the live-test service account,
  whose project was made in the old organization (register item O34f; the
  run before `ofr-11160002-a4c3` was refused so); the fixtures' own grants
  go to `OFFRAMP_MEMBER`. Its organization policies need
  `orgpolicy.policyAdmin` (`OFFRAMP_ORG_POLICIES=false` leaves them out).
- platform-project: the project is created and linked to billing, then the
  provider's deletion of its default network fails before the project's
  Compute Engine API is usable (O34e: `compute.skipDefaultNetworkCreation`
  on the folder).

Rerun 2026-11-18, after the grants of tf-infra #25 and #30 (Deny Admin on
the organization, the `live-tests` folder's policy), acting as
`live-tests@hfl-testing-8347a2` through `offload run --impersonate`:

- folders (run `ofr-11181651-08f6`, organization policies left out, the
  hierarchical firewall policy in): exit 0, 14 resources applied,
  `terraform plan` empty, 5 calls graduated, Bundt's server-side dry run
  and `terraform plan` after adoption both empty, the abandoned objects
  left every resource, `terraform destroy` clean, nothing named
  `ofr-11181651-08f6` left. **Not the clean no-op of `ofr-11160002-a4c3`**:
  the export refused all 6 folders (`accessNotConfigured`), only 6 of 14
  objects reached `Ready` `UpToDate`, 7 ended `UpdateFailed` and 6
  `DependencyNotReady`. The run's artifacts were not kept, so the exact
  API error was not read; the cause is taken to be that the Cloud
  Resource Manager, Service Usage and Cloud Billing APIs are not enabled
  in `live-tests@`'s home project (tf-infra's `hfl-testing` lists them;
  `gcloud services list` shows only Compute Engine, IAM, IAM Credentials
  and Logging there), which a Config Connector call made as that account
  is charged to. Until they are, the earlier run stays the no-op proof.
  Its audit-log row (15 `CreateSink` in `hfl-dev-514205`) was the
  config-connector-extensions log-sink tests running as the same account
  at the same time, not the off-ramp.
- platform-project (run `ofr-11181736-8aaf`): blocked at `terraform apply`
  before anything was made: 403 `Cloud Billing API has not been used in
  project 50660598795` (the same project) when the provider checked the
  billing account. `terraform destroy` ran clean, nothing left. The
  folder's `compute.skipDefaultNetworkCreation` policy (O34e) is in place
  and is untested until the APIs are on.

gke has not run live yet. The non-live shapes are validated
(`task offramp-validate`) and converted offline from synthetic states.

## The converter

```sh
go build -o ../bin/offramp ./cmd/offramp

# From a state. --kcc is a Config Connector checkout at the release the
# cluster runs (its config/crds/resources and config/servicemappings).
# --root graduates the root module's calls to catalog charts: it reads the
# module blocks (catalogtool convert), takes each call's source and version
# from .terraform/modules/modules.json when terraform init has run (or
# --modules-json), renders the chart, and compares the render with the
# objects converted from the state.
../bin/offramp state --kcc ~/src/k8s-config-connector --state terraform.tfstate \
  --root path/to/root-module --tfvars path/to/terraform.tfvars \
  --out out --name shared-vpc --team platform --env prod --target infra

# A synthetic state for a root module that cannot be applied: Terraform
# plans it offline, repeatedly; only what the cloud assigns is made up.
../bin/offramp synth --root fixtures/folders --tfvars folders.tfvars --out folders.tfstate

# From a scan: the assets, the names to export, Config Connector's export,
# then the same grouping (no graduation: a scan has no module calls).
hack/scan.sh --scope folders/123 --billing-project my-project >assets.json
../bin/offramp names --assets assets.json >names.txt
while read -r n; do config-connector export "$n" --iam-format policymember --output "export/$(echo "$n" | md5sum | cut -c1-8).yaml"; done <names.txt
../bin/offramp krm --krm export --out out-scan

# What one has that the other lacks, field by field.
../bin/offramp compare out/krm/all.yaml out-scan/krm/all.yaml
```

The output (`--out`) holds:

- `krm/all.yaml`: every object, frozen.
- `krm/adopt.yaml`: what Bundt delivers on the import. That is the chart's
  render for each graduated call and the frozen objects for every other
  call. Every object carries `deletion-policy: abandon` and
  `state-into-spec: absent`.
- `packages/<package>/manifests/`: the frozen Packages.
- `graduation/<component>/`: the values, the render, the state's objects and
  their differences.
- `bundt/prereqs.yaml`, `bundt/catalog.yaml` and `bundt/record.yaml`. A
  graduated call is a Template component naming the chart's Package, with
  the call's literal arguments as inputs. Its per-environment arguments are
  the Instance's `spec.components.<c>.values`.
- `report.md`: what became what, the "Module calls and catalog charts"
  table, and every finding.

IAM converts as the catalog's charts render it (`docs/design.md`, "IAM"):
`google_<x>_iam_member` becomes an `IAMPolicyMember`;
`google_<x>_iam_binding`, authoritative for its role (and condition),
becomes one `IAMBinding` (config-connector-extensions,
`iam.gcp.hfl.dev/v1beta1`), exact as Terraform's, on the 12 kinds that kind
takes; on any other (a subnet) an `IAMPartialPolicy` with that one binding,
with a note that members added out of band are no longer removed. A
binding with no members is refused (an `IAMBinding` needs one). The
comparison counts each binding as one grant per member and role, so a
chart's `IAMPartialPolicy` and the state's `IAMBinding` for the same grants
match.

### Graduation: the equivalence check

For each call whose module a chart ports:

1. The call's arguments become the chart's values unchanged.
2. The import adds `deletion_policy: ABANDON`. Where the chart names a
   resource by a cloud-assigned ID (a folder's number, a hierarchical
   firewall policy's ID), it also adds `resource_ids` from the state. Where
   the call needs an alpha kind, it adds `allow_alpha: true`.
3. Another call's output (`module.vpc.id`) becomes
   `${components.vpc.outputs.id}`. A function applied to one becomes CEL
   (`trimprefix(module.root.id, "folders/")`). To check the render, the
   expression is evaluated against the producer's chart outputs. Those
   outputs are computed from its render, with each object's status taken
   from the state.
4. The chart is rendered with `helm template` and compared with the
   resource-level conversion, resource by resource, with references
   compared by name.
5. A field the render leaves out does not count as a difference. With
   `state-into-spec: absent`, Config Connector neither sets nor changes it,
   and the report lists it as left to the cloud.
6. Only a call with no differences graduates. Any other call keeps its
   frozen Package and lists why. A graduated call that reads a frozen
   call's output gets the resolved value instead.

`go test ./...` runs this for every fixture (`TestEquivalence`): against the
live shared-vpc state, and against synthetic states for all four shapes.

## Terraform and OpenTofu states

Every command that takes a state (`state`, `terragrunt`, `compose`) reads
the same forms, from either tool:

| Form | From | Notes |
| --- | --- | --- |
| Raw state, format 4 | Terraform 0.12 and later, every OpenTofu release (`terraform.tfstate`, `state pull`) | the tool is told by the providers' registry host (`registry.opentofu.org` or `registry.terraform.io`) |
| `show -json` (`format_version` 1.x) | `terraform show -json`, `tofu show -json` | the documented, versioned form; carries no per-instance dependencies (a finding says so) |

Refused, each with what to run instead: an OpenTofu-encrypted state
(1.7 and later; `tofu state pull` or `tofu show -json` in the root
decrypts it with its own keys, which the off-ramp never holds), format 3
or older (Terraform 0.11), a format newer than 4, and a `show -json` major
version other than 1. A deposed instance (left by `create_before_destroy`)
is not adopted and is listed. A sensitive value is removed by its exact
path, so the rest of its block still converts (`master_auth` keeps
`client_certificate_config` without `client_key`). The fixtures in
`internal/offramp/testdata/stateformats/` were written by Terraform
1.17.2 and OpenTofu 1.10.7 (`stateformats_test.go`).

## Composing charts: `offramp compose`

Where no catalog chart ports a call's module, `offramp compose` authors
one from the state instead of freezing it: a ChartSpec, a catalog-shaped
chart with the standard placement values and sizing knobs, and the same
zero-diff proof. See `docs/compose.md`.

## Terragrunt estates

`offramp terragrunt` reads a Terragrunt estate: every `terragrunt.hcl`
under `--estate` that no other unit includes is a unit, a Terraform root
module with its own state.

```sh
../bin/offramp terragrunt --kcc ~/src/k8s-config-connector --estate live/prod \
  --states pulled-states --out out --name prod --team platform --env prod
```

**Nothing runs.** Neither Terragrunt nor Terraform: each file is parsed
with HCL and evaluated with the functions that need only the files.

- **Read.** `include` (labelled or not, `expose`, `merge_strategy`
  shallow, deep or no_merge) with the included file evaluated in the
  unit's context, and `locals`. `inputs` merged over the included ones.
  `terraform { source }` (`tfr://`, `git::`, GitHub, the registry, a local
  path with Terragrunt's `//`). `dependency` and `dependencies`.
  `remote_state` and `generate`.
- **Functions.** `find_in_parent_folders`, `read_terragrunt_config`,
  `path_relative_to_include` and `path_relative_from_include`,
  `get_terragrunt_dir`, `get_parent_terragrunt_dir`, `get_repo_root` and
  its two path functions, `read_tfvars_file`, and `get_env`, which reads
  `--getenv NAME=VALUE`, else the default, and notes which. Also
  Terraform's functions that need no run, including `file`,
  `yamldecode` and `jsondecode`.
- **Never guessed.** `run_cmd`, `sops_decrypt_file`, the `get_aws_*`
  functions, `get_platform`, `get_terraform_command`, `timestamp`,
  `uuid`, and a `get_env` with no default and no `--getenv` cannot be
  evaluated from the files. Neither can a function, condition or loop over
  a dependency's output. The input, local or source that needs one is left
  out, and an `unevaluated` finding names the expression and why. A local
  that fails fails only what reads it.
- **Listed, not run.** `generate` blocks (the provider and backend files)
  are listed with their paths. Hooks and `extra_arguments` are noted.
  `mock_outputs` are never used.
- **Where the state is.** `remote_state` gives each unit's state location
  (`gs://<bucket>/<prefix>/default.tfstate` for gcs). The offramp reads
  the state from `--states DIR` (`DIR/<unit path>/terraform.tfstate`, from
  `terragrunt state pull`) or `--state UNIT=FILE`. A unit with no state
  is a `gap` finding that says where its state lives.

**Two cases.** A unit's source is one of two things:

1. **An open-source module the catalog tracks**, such as
   `tfr:///terraform-google-modules/network/google//modules/firewall-rules?version=18.3.1`.
   The module is the unit's root. The unit is converted as one call of
   that module, its inputs the arguments. Like a module call, it graduates
   to the module's chart when the render matches the unit's state. An
   input the chart has no value for is noted: Terraform ignores an input
   the module does not declare, and the chart ports every variable.
2. **The team's own root module**, a wrapper: a local path, or a remote
   source mapped to a checkout with `--source-map URL=DIR`. It may call
   open-source modules (Cloud Foundation Fabric's project module,
   terraform-google-modules' beta-private-cluster). The offramp descends
   into it. Each module call inside is converted and checked against its
   chart, as in a plain root module, with the unit's inputs as the
   wrapper's variables. The calls become components `<unit>-<call>`. The
   wrapper's own resources (no module makes them) stay a frozen Package,
   component `<unit>`, each listed as an `unmapped` finding.

A remote source that is neither is a `gap`: its resources stay one frozen
Package.

**Between units.** The estate is one Template. A `dependency` output
becomes a reference to the producing component's output:
`dependency.vpc.outputs.name` is `${components.vpc.outputs.name}`. For a
wrapper, the reference goes through the wrapper's `output` block to the
call output it passes on. An output a wrapper computes itself has no
component, so it is reported. `dependency` and `dependencies` become
`dependsOn`. Each object's `offramp.hfl.dev/source` is
`<unit path>:<Terraform address>`, the state the hand-off removes it from.

**The fixtures** (`fixtures-terragrunt/`, tested by `TestTerragruntEquivalence`):

| Unit | Source | Case | Result |
| --- | --- | --- | --- |
| `network/vpc` | fabric `net-vpc` v60.0.0 over `git::` | module | graduates to `net-vpc` |
| `network/firewall-rules` | tgm `firewall-rules` 18.3.1 over `tfr://`; `dependency.vpc.outputs.name` | module | graduates to `firewall-rules`, `network_name: ${components.vpc.outputs.name}` |
| `network/network-firewall-policy` | tgm `network-firewall-policy` 18.3.1 over `git::`; `dependency.vpc.outputs.id` | module | graduates to `network-firewall-policy` |
| `gke/gke-platform` | `modules//gke-platform`, a wrapper of fabric `net-vpc` and tgm `beta-private-cluster` 47.0.0, plus a firewall rule, a service account and a project grant of its own | wrapper | `network` graduates; `cluster` stays frozen (`datapathProvider`, as in `fixtures/gke`); the three own resources are `unmapped` |
| `platform/project` | `fixtures//platform-project`, a wrapper of fabric `project`, `iam-service-account`, `gcs` and `secret-manager` | wrapper | as `fixtures/platform-project`: five calls graduate, `project` stays frozen (`ServiceIdentity`) |

The network units' states are the live shared-vpc run's, split per unit.
The others are synthetic (`internal/offramp/testdata/terragrunt/README.md`).
`TestTerragruntMatchesRenderJSON` checks the static reading against
`terragrunt render-json`.

## Tests

- `task offramp-test`: gofmt, vet and `go test`, including the equivalence
  check of every fixture. It needs helm.
- `task offramp-validate`: checks the fixtures on an offload VM and keeps
  each fixture's `modules.json`.
- `task offramp-live SHAPE=shared-vpc`: runs the live proof in the
  live-test project (`GCP_PROJECT`) as `GCP_CCE_LIVE_SA`, adopting
  `krm/adopt.yaml`; the other shapes need the grants the design lists.

Each takes `FIXTURES`, the directory of root modules relative to the
repository (default `offramp/fixtures`).
