# container-registry

> **Deprecated for new setups.** gcr.io repositories are legacy: this chart
> is supported for organizations still on gcr.io. New setups should use a
> native Artifact Registry repository chart
> ([`artifact-registry`](../artifact-registry)) and Artifact Registry roles.

A Container Registry host as an Artifact Registry repository with gcr.io
domain support and its IAM, as Config Connector objects. A port of Cloud
Foundation Fabric's
[`modules/container-registry`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/container-registry)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-container-registry` (`package.yaml`). Checked against Config Connector
v1.160.0.

Google's Container Registry is shut down. Its successor for a `gcr.io` host
is an Artifact Registry repository with gcr.io domain support, which Google
names after the host and places in the host's multi-region. So the chart
does not build a Container Registry kind: it maps Terraform's
`google_container_registry` to Config Connector's
`ArtifactRegistryRepository` (config-connector-extensions
`docs/design.md`, decision 30; `docs/design.md` here, "Container
Registry").

| `location` | Repository (`resourceID`) | Artifact Registry location |
| --- | --- | --- |
| empty (the default host) | `gcr.io` | `us` |
| `US` | `us.gcr.io` | `us` |
| `EU` | `eu.gcr.io` | `europe` |
| `ASIA` | `asia.gcr.io` | `asia` |

The table is Google's
([Set up gcr.io repositories](https://cloud.google.com/artifact-registry/docs/transition/setup-gcr-repo)).
`location` is read in any case (`us` is `US`).

Pushes and pulls through `gcr.io` reach the repository only once the
project's gcr.io traffic is redirected to Artifact Registry
(`gcloud artifacts settings enable-upgrade-redirection --project=<project>`).
That is a project setting Config Connector has no kind for; the chart does
not set it.

## Example

```yaml
project_id: my-project
location: ""
iam:
  roles/artifactregistry.reader: ["user:foo@example.com"]
```

renders an `ArtifactRegistryRepository` `gcr.io` in `us` (format `DOCKER`)
and its `IAMPartialPolicy` (`tests/golden/basic.yaml`;
`tests/golden/full.yaml` is an `eu.gcr.io` registry with labels and
`ABANDON`).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names`, `namePrefix` | not fabric variables: Google fixes a gcr.io repository's name by its host, so `names` takes no key and `namePrefix` changes nothing (`tests/name-prefix.yaml`) |
| `labels`, `tags` | Kubernetes labels on every object; unprefixed ones and `tags` are Google Cloud labels on the repository |
| `project_id` | the repository's project (`cnrm.cloud.google.com/project-id`) |
| `location` | the repository's name and location, per the table above; anything but `US`, `EU`, `ASIA` or empty is refused |
| `iam` | one `IAMPartialPolicy` on the repository (`docs/design.md`, "IAM": the listed members are kept, others left alone). Fabric bound these on the registry's Cloud Storage bucket; the roles are now Artifact Registry roles, and five Cloud Storage roles are translated (below) |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the repository and its images |

### Cloud Storage roles in `iam`

A Cloud Storage role cannot be granted on an Artifact Registry repository.
So that a module call's `iam` converts one to one, the chart translates
these five (the builder, 2026-11-15, CRREG-1; `docs/design.md`,
decision 13). Google gives no role-for-role table; this one is ours.

| Cloud Storage role | Artifact Registry role |
| --- | --- |
| `roles/storage.objectViewer` | `roles/artifactregistry.reader` |
| `roles/storage.legacyBucketReader` | `roles/artifactregistry.reader` |
| `roles/storage.legacyBucketWriter` | `roles/artifactregistry.writer` |
| `roles/storage.legacyBucketOwner` | `roles/artifactregistry.repoAdmin` |
| `roles/storage.admin` | `roles/artifactregistry.repoAdmin` |

Roles that translate to one Artifact Registry role, or to one `iam` also
lists as is, share one binding, each member once
(`tests/storage-merge.values.yaml`). Each converted role gives a note,
written as a `# NOTE` comment at the top of the rendered
`IAMPartialPolicy` and printed by Helm on install (`templates/NOTES.txt`):
it names the role and its translation, says gcr.io repositories are
legacy, and that new setups should use a native Artifact Registry
repository with Artifact Registry roles directly. Bundt's renderer drops
Helm's NOTES.txt, so for Bundt the note is the comment in the rendered
object.

`legacyBucketOwner` and `storage.admin` to `repoAdmin` is a judgment call:
`repoAdmin` reads, writes and deletes images in the repository but does
not manage the repository's IAM, which both bucket roles could on the
bucket. Review those bindings.

Refused, in part:

| Variable | Why |
| --- | --- |
| `iam` with any other `roles/storage.*` role | a Cloud Storage role cannot be granted on an Artifact Registry repository, and only the five above translate. The message names the roles to use: `roles/artifactregistry.reader` to pull, `.writer` to push, `.repoAdmin` to also delete (`tests/storage-role.fail.yaml`) |

## Outputs

| Output | Value |
| --- | --- |
| `id` | `projects/<project>/locations/<us\|europe\|asia>/repositories/<host>`, from the inputs. Fabric's `id` was the registry's Cloud Storage bucket (`artifacts.<project>.appspot.com`); there is no bucket now, so it is the repository's full resource name |

## From Terraform

A `module "x" { source = ".../modules/container-registry" ... }` block's
arguments become the values unchanged (`examples/container-registry/`).
The five Cloud Storage roles above are translated with a note; any other
Cloud Storage role in `iam` is refused with the Artifact Registry role to
use. The `id` output changes meaning: it was the bucket's name and is now
the repository's resource name ("Outputs"). A project that already set up its gcr.io repository
(by hand, or with `gcloud artifacts docker upgrade migrate`) keeps it: Config
Connector acquires the repository of that name and location
(`docs/design.md`, "From Terraform"). Images still in a Container Registry
bucket are not copied by the chart; Google's migration tool copies them.

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/container-registry`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/container-registry)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml) records
the repository, path, release, commit, licence and the date last synced, and
the module is listed in [`upstream/modules.yaml`](../../../upstream/modules.yaml)
with every other module the catalog tracks. To take a newer release, see
[`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=container-registry`: render cases `basic`, `full`, `us`,
`asia`, `mock`, one per translated Cloud Storage role
(`storage-object-viewer`, `storage-legacy-bucket-reader`,
`storage-legacy-bucket-writer`, `storage-legacy-bucket-owner`,
`storage-admin`) and `storage-merge`, each validated against Config Connector v1.160.0's CRD
schemas, and the refusals in `tests/*.fail.yaml`.

Not yet applied to the fake cloud or live: `tests/mock.values.yaml` is ready
for `task test-mock MOCK_CHARTS=container-registry`.

## Changes

- **0.2.0** (2026-11-15): five Cloud Storage roles in `iam` are
  translated to Artifact Registry roles, each with a note; any other is
  still refused. A deprecation line: gcr.io repositories are legacy (the
  builder, CRREG-1).
- **0.1.0** (2026-11-15): the port, as an Artifact Registry gcr.io
  repository (decision 30).
