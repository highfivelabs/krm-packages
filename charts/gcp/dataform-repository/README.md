# dataform-repository

A Dataform repository, optionally linked to a remote Git repository, and
IAM on it, as Config Connector objects. A port of Cloud Foundation
Fabric's
[`modules/dataform-repository`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/dataform-repository)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-dataform-repository` (`package.yaml`). Checked against Config
Connector v1.160.0's `DataformRepository` and config-connector-extensions'
`IAMBinding` (`iam.gcp.hfl.dev/v1beta1`) and `IAMMember`
(`iam.gcp.hfl.dev/v1beta1`), at `81c3f9b5`: Config Connector's own IAM
kinds cannot reference a Dataform repository.

## Example

```yaml
project_id: my-project
name: my-remote
region: europe-west1
service_account: dataform@my-project.iam.gserviceaccount.com
remote_repository_settings:
  url: https://github.com/example/dataform.git
  secret_version: projects/my-project/secrets/my-token/versions/1
iam:
  roles/dataform.editor: ["group:data-eng@example.com"]
```

renders the `DataformRepository` `my-remote` linked to the GitHub
repository on `main`, reading its token from the secret version, and an
`IAMBinding` for the editor role (`tests/golden/remote.yaml`,
`tests/golden/basic.yaml`).

## Values

The module's variables, with the same names, types and defaults. The
module has eight; the chart carries all eight, `remote_repository_settings`
in part.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.repository` sets the repository's name in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the repository's name (unless set under `names`) starts with `<namePrefix>-`; empty changes nothing |
| `labels`, `tags` | not fabric variables: Kubernetes labels on every object. `DataformRepository` has no labels field |
| `project_id` | every object's project annotation, the repository's `projectRef` |
| `region` | the repository's `region` |
| `name` | the repository's name (`resourceID`) |
| `service_account` | the repository's `serviceAccountRef.external` (an email) when set |
| `remote_repository_settings` | `gitRemoteSettings`: `url`, `branch` (default `main`) as `defaultBranch`, and `secret_version` as `authenticationTokenSecretVersionRef.external`. `secret_name` is accepted and not sent, as fabric does; `token` is refused (below) |
| `iam` | one `IAMBinding` per role, exact |
| `iam_bindings` | one `IAMBinding` per entry, exact, with its condition |
| `iam_bindings_additive` | one `IAMMember` per entry (additive), with its condition |
| `allow_alpha` | not a fabric variable: no kind of this chart is alpha any more (`IAMMember` is `v1beta1` since config-connector-extensions `81c3f9b5`); kept so components that set it still render, it changes nothing |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the repository (not the IAM) |

Refused:

- `remote_repository_settings.token`: a token given here would be in the
  rendered manifest, and fabric's module does not send it either. Put it
  in Secret Manager and set `secret_version` to its version.

Checks: with `remote_repository_settings`, `url` is required (the kind and
the API require it), and `secret_version` must be a secret version's full
name (`projects/<project>/secrets/<secret>/versions/<version>`). fabric
passes `secret_version` as given, default `v1`, which the API refuses; the
chart stops the render instead, at plan time.

IAM is on the repository by its full name
(`projects/<project>/locations/<region>/repositories/<name>`), so the IAM
objects can be deleted after the repository's.

## Outputs

None: fabric's module declares none.

## Differences from the module

- **`secret_version`** must be a full name (above).
- **`secret_name` and `token`**: fabric's variable has them and its
  module uses neither; the chart accepts `secret_name` and ignores it, and
  refuses `token`.

## From Terraform

A `module "x" { source = ".../modules/dataform-repository" ... }` block's
arguments become the values unchanged (without `token`). The repository keeps its
name, so it adopts what Terraform made (`docs/design.md`, "From
Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/dataform-repository`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/dataform-repository)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=dataform-repository`: render cases `basic`, `remote`,
`full` (every variable), `names` and `mock`, each validated against
Config Connector's CRDs and those in `crds/config-connector-extensions`,
and the refusals in `tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`): a repository in `us-central1` and a role on it (`iam`). On 2026-11-15 (offload VM `off-catalog-1115092825-18945`, config-connector-extensions `384d07c0`, whose fake cloud serves any `getIamPolicy` path) the repository and its `IAMBinding` became Ready, the outputs evaluated (none) and both were deleted. The remote repository settings, `iam_bindings` (with conditions) and the `IAMMember`s are only render-tested.

Not applied live.

## Changes

- **0.2.0** (2026-11-15): `IAMMember` renders
  `iam.gcp.hfl.dev/v1beta1` without `allow_alpha` (config-connector-extensions
  `81c3f9b5`).
- **0.1.0** (2026-11-15): the port: the repository and IAM on it.
