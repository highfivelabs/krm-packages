# cloud-deploy

A Cloud Deploy delivery pipeline, its targets, automations and deploy
policies, and IAM on the pipeline and targets, as Config Connector and
config-connector-extensions objects. A port of Cloud Foundation Fabric's
[`modules/cloud-deploy`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/f706bc3b6166787df01402f73a96f1de57097477/modules/cloud-deploy)
at v60.0.0 (`f706bc3b`), Apache-2.0, Copyright Google LLC. Bundt Package
`gcp-cloud-deploy` (`package.yaml`). Checked against Config Connector
v1.160.0's `CloudDeployDeliveryPipeline` and config-connector-extensions'
`CloudDeployTarget`, `CloudDeployAutomation`, `CloudDeployDeployPolicy`
(`clouddeploy.gcp.hfl.dev/v1beta1`), `IAMBinding`
(`iam.gcp.hfl.dev/v1beta1`) and `IAMMember` (`iam.gcp.hfl.dev/v1beta1`),
at `81c3f9b5`. Config Connector's own `CloudDeployTarget` is `v1alpha1`,
it has no automation or deploy policy kind, and its IAM kinds cannot
reference a pipeline or a target.

## Example

```yaml
project_id: my-project
region: europe-west1
name: deployment-pipeline
targets:
  - name: dev-target
    description: Dev Target
    profiles: [dev]
    cloud_run_configs:
      automatic_traffic_control: true
  - name: prod-target
    description: Prod Target
    profiles: [prod]
    require_approval: true
    cloud_run_configs:
      automatic_traffic_control: true
```

renders the `CloudDeployDeliveryPipeline` `deployment-pipeline`, a serial
pipeline with a standard stage per target, and the two Cloud Run
`CloudDeployTarget`s, the second requiring approval
(`tests/golden/basic.yaml`, fabric's README example).

## Values

The module's variables, with the same names, types and defaults.

| Variable | Renders |
| --- | --- |
| `names` | not a fabric variable: `names.pipeline` sets the pipeline's name in full |
| `namePrefix` | not a fabric variable: an optional convenience; when set, the pipeline's name (unless set under `names`), every target the chart makes and every deploy policy start with `<namePrefix>-`, and the stages and multi-targets that name those targets follow. Automations live in the pipeline and keep their keys; a target the chart does not make (`create_target: false`), and a target ID in an automation's rules or selector, are as given |
| `labels` | fabric's variable (the pipeline's labels), with the catalog's meaning: labels on every object; an unprefixed one is a Google Cloud label too |
| `tags` | not a fabric variable: cloud labels on every object |
| `project_id`, `region` | the pipeline's project and location, and every other object's unless it sets its own |
| `name`, `description`, `annotations`, `suspended` | the pipeline's |
| `targets` | a stage of the serial pipeline per target not `exclude_from_pipeline`, in order: `profiles`, `delivery_pipeline_deploy_parameters`, and a `STANDARD` strategy (`verify`, `predeploy_actions`, `postdeploy_actions`) or a `CANARY` one (`deployment_percentages`, or `custom_canary_phase_configs` as custom phases; `cloud_run_configs`' traffic control and revision tags). And a `CloudDeployTarget` per target with `create_target` (the default): `annotations`, `description`, `labels`, `target_deploy_parameters`, `require_approval`, `execution_configs_usages` and `_timeout`, `multi_target_target_ids`, and `cloud_run_configs` as the Cloud Run location. Each target's own `iam`, `iam_by_principals`, `iam_bindings` and `iam_bindings_additive` go on that target, made here or not |
| `automations` | one `CloudDeployAutomation` per key, its ID the key, on the pipeline: a rule per rule set (advance rollout, promote release, repair rollout with its retry and rollback phases, timed promote release), the selector (default every target), `service_account` (required, below), `suspended`, `annotations`, `labels`, `description` |
| `deploy_policies` | one `CloudDeployDeployPolicy` per key: a rollout restriction per key of `rollout_restrictions` (actions, invokers, time zone, weekly and one-time windows), the selectors (default every pipeline), `suspended`, `annotations`, `labels`, `description` |
| `iam`, `iam_by_principals` | one `IAMBinding` per role on the pipeline, exact, the two merged as fabric merges them |
| `iam_bindings` | one `IAMBinding` per entry on the pipeline, exact, with its condition |
| `iam_bindings_additive` | one `IAMMember` per entry on the pipeline (additive), with its condition |
| `allow_alpha` | not a fabric variable: no kind of this chart is alpha any more (`IAMMember` is `v1beta1` since config-connector-extensions `81c3f9b5`); kept so components that set it still render, it changes nothing |
| `deletion_policy` | not a fabric variable; `ABANDON` keeps the pipeline, targets, automations and deploy policies (not the IAM) |

Refused:

- An automation without `service_account`: fabric then reads the
  project's Compute Engine default service account, whose email needs the
  project number, which a render does not have. Set it.

Checks, as fabric's validations: at least one rule per automation; at
least one window per rollout restriction; a selector's type is
`DELIVERY_PIPELINE` or `TARGET`; a target's strategy is `STANDARD` or
`CANARY`, and a canary's percentages ascend between 0 and 99; the
pipeline's name and description. And where the API and the kinds require
a field fabric makes optional: a rollout restriction's `time_zone`, all
four of a one-time window's dates and times, `execution_configs_usages`
with a timeout, and a retry or a rollback in a repair rule. Numbers fabric
types as strings (times, dates, retry attempts, phase percentages) must be
numbers.

IAM is on the pipeline and targets by their full names
(`projects/<project>/locations/<region>/deliveryPipelines/<name>`,
`.../targets/<target>`), so the IAM objects can be deleted after theirs.

## Outputs

| Output | From |
| --- | --- |
| `pipeline_id` | `projects/<project>/locations/<region>/deliveryPipelines/<name>`, from the rendered pipeline |
| `target_ids` | the IDs of the targets the chart makes, sorted |
| `automation_ids` | the automations' IDs, sorted |
| `deploy_policy_ids` | the deploy policies' IDs, sorted |

## Differences from the module

- **`promote_release_rule.destination_phase`** is sent as given. fabric
  sends `destination_target_id` as the destination phase too, which the
  API takes for a phase name.
- **Custom canary phases**: fabric's resource reads `percentage` and
  `verify` from each phase, which the variable does not have (it has
  `deployment_percentage`), so a custom canary fails in Terraform; the
  chart sends `deployment_percentage` as the phase's percentage and the
  target's `verify`, and the custom phases instead of the percentages (the
  API takes one or the other).
- **Fabric's Compute Engine default service account** for automations is
  refused (above).

## From Terraform

A `module "x" { source = ".../modules/cloud-deploy" ... }` block's
arguments become the values unchanged, with `service_account` set on
every automation. The
pipeline, targets, automations and policies keep their names, so they
adopt what Terraform made (`docs/design.md`, "From Terraform").

## Upstream

Ported from Cloud Foundation Fabric's
[`modules/cloud-deploy`](https://github.com/GoogleCloudPlatform/cloud-foundation-fabric/tree/v60.0.0/modules/cloud-deploy)
at v60.0.0 (`f706bc3b`, Apache-2.0). [`upstream.yaml`](upstream.yaml)
records the repository, path, release, commit, licence and the date last
synced, and the module is listed in
[`upstream/modules.yaml`](../../../upstream/modules.yaml). To take a newer
release, see [`UPSTREAM.md`](../../../UPSTREAM.md), "Moving the pins".

## Tests

`task test CHART=cloud-deploy`: render cases `basic` (fabric's README
example), `full` (every variable), `names` and `mock`, each validated
against Config Connector's CRDs and those in
`crds/config-connector-extensions`, and the refusals in
`tests/*.fail.yaml`.

Fake cloud (`task test-mock`, `tests/mock.values.yaml`): a pipeline, a Cloud Run target, an automation, a deploy policy, and a role on the pipeline and on the target. On 2026-11-15 (offload VM `off-catalog-1115095124-18431`, config-connector-extensions `384d07c0`, whose fake cloud serves any `getIamPolicy` path) all six objects became Ready, the outputs evaluated and all six were deleted. Canary strategies, multi-targets, the other automation rules, one-time windows, `iam_bindings` (with conditions) and the `IAMMember`s are only render-tested.

Not applied live.

## Changes

- **0.2.0** (2026-11-15): `IAMMember` renders
  `iam.gcp.hfl.dev/v1beta1` without `allow_alpha` (config-connector-extensions
  `81c3f9b5`).
- **0.1.0** (2026-11-15): the port: the pipeline, targets, automations,
  deploy policies and IAM.
