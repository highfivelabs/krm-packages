# Composing charts from a Terraform state

`offramp compose` is the Terraform flavor of the Bundt chart composer
(Linear project "Bundt: Chart Composer", BND-62). The composer has three
ways in. All three produce a ChartSpec, and from there the steps are the
same:

| Input | Ticket | What it starts from |
| --- | --- | --- |
| Intent | BND-61 | an agent's proposal from a stated need |
| Live working set | BND-58 | resources an agent deployed piecemeal in dev |
| Existing chart | BND-59 | a chart plus a change request |
| **Terraform state** | **BND-62** | **what Terraform built, read from its state** |

## Where it sits in the off-ramp

The off-ramp (`offramp state`) already adopts a Terraform estate with no
change to the cloud. Each module call gets one of two outcomes:

1. **Graduated:** a catalog chart ports the call's module, and its render
   matches the state. The call becomes an Instance of that chart.
2. **Frozen:** otherwise the call becomes a frozen Package. It is
   correct and adoptable, but it is a snapshot, not a chart anyone can
   reuse or change through values.

`compose` replaces the frozen outcome with a third one:

3. **Composed:** a new chart authored from the state, with the standard
   values and the same zero-diff proof. It is a candidate for the team's
   catalog, or for the public catalog when the module is common.

So the order for a call is: graduate if a catalog chart fits, else
compose, and freeze only if composing can't prove zero diff. Teams with
no modules at all (resources straight in the root module) get a chart for
their root module.

## Steps

```
state ─► convert ─► normalize ─► classify ─► choose values ─► ChartSpec ─► compile ─► verify
         (offramp    (refs inside  (stateful,   (placement,                (chart dir)  (lint, zero diff,
          state)      the chart     identity     knobs, adoption)                         collisions)
                      by name)      fields)
```

- **Convert** is the off-ramp's existing state → Config Connector
  conversion, unchanged.
- **Normalize** drops Terraform provenance. A reference to an object
  inside the chart, by name or by its resource ID (`folders/123`),
  becomes a `name:` ref, so a renamed instance stays wired. A
  reference outside the chart stays `external` and is listed as a
  candidate input.
- **Classify** marks stateful kinds (always `abandon`, whatever the
  values say) and each resource's identity fields (changing them
  recreates the resource; the recreate guard, BND-57, will read these).
- **Choose values** is deterministic for now:
  - **Placement** (always exposed): `projectId`, `region`, `network`,
    `subnetwork`, `codeName`, `environment`, `labels`. Their literals are
    replaced in every string at word boundaries, most specific first.
    Network and subnetwork are matched only after `networks/` and
    `subnetworks/` or as a network's own name, because a network and
    its subnet often share a name. Zones become `region` plus their
    letter.
  - **Knobs:** known sizing fields per kind (`knobFields`: machine type,
    disk size and type, node counts, release channel, CIDR range, SQL
    tier, …). The value is named per resource when the chart has more
    than one of that kind (`appsMachineType`).
  - **Override:** `names`, full names by resource ID (see "Names"
    below).
  - **Adoption:** `deletionPolicy`, and `resourceIds` for cloud-assigned
    IDs (a folder's or a hierarchical firewall policy's number). These
    bind this adoption to these resources and are empty for a new
    instance.
- **Compile** writes a catalog-shaped chart: `Chart.yaml` with
  provenance annotations, `values.yaml` in tiers, a closed
  `values.schema.json`, `package.yaml` with CEL outputs, one template per
  resource, a README, and `chartspec.yaml` (helmignored). The chart can
  be regenerated from that spec whenever the compiler improves.
- **Verify:**
  - `helm lint`
  - render with defaults and compare with the state's objects (spec,
    labels and project), with zero differences required
  - the collision test: render three instances (new codename, network and
    subnetwork, no adopted IDs; a new project when the chart makes the
    project) and list any identity they share

A collision is reported, not failed. Some are real and correct: two
instances granting on the same parent folder, or on another call's
subnet, really do share that policy.

## The codename

The codename (the `codeName` value, formerly "moniker") is the most
common name prefix (at a `-`) across what the
chart makes, after stripping a leading project ID. If there is no shared
prefix, it is the name of the first resource (the module's main one).
Names that the converter built around a cloud-assigned ID
(`n100000000765`) don't vote. `--code-name` overrides it. Bundt mints
short codenames (8 characters or fewer) for new instances; an imported
codename that is longer is flagged.

## Names: convention by default, an escape hatch always

The codename convention keeps instances in one project from colliding,
but a chart a customer brings, or one composed from an import, often
follows the organization's own naming instead, and a generic chart is
rarely expressive enough for every organization. So every name is
overridable:

```yaml
names:
  containercluster-ofr-test-std: acme-prod-gke   # by resource ID
```

A resource's `metadata.name` (and a `resourceID` equal to it) holds
`${name.<id>}` in the ChartSpec, defaulting to `nameDefault`, the name
the conventions gave. Every reference to it inside the chart holds the
same binding, so overriding one name rewires its dependents: the node
pools' `clusterRef` follows the cluster, a folder's grants follow the
folder. The values schema lists the resource IDs a name can be given for,
so a typo is refused. With no override, names render exactly as before
and the zero-diff proof is unchanged. Names Bundt cannot vary (fixed by
default) are listed as findings, with the override that would fix them.

## Terraform and OpenTofu

Composing works the same from either tool's state, raw or `show -json`
(the README's "Terraform and OpenTofu states"). The ChartSpec's source
records `tool` (`terraform` or `opentofu`), `form` (`state` or
`show-json`) and the tool version that last wrote the state, and the
chart carries them as `hfl.dev/source-tool`, `hfl.dev/source-form` and
`hfl.dev/source-tool-version`. `TestComposeFromOpenTofu` composes the gke
charts from OpenTofu 1.10.7's `show -json` and proves them with zero diff.

## ChartSpec

`schema/chartspec.v1alpha1.json` is the first concrete draft of BND-51.
A bound field holds `${values.<name>}`: as the whole string it takes the
value with its type, and inside a longer string it interpolates. A labels
map may hold `"$merge": "${values.labels}"`. A cloud-assigned ID holds
`${resourceIds.<resource id>}`. When the composer service exists, the
spec and compiler move there and this command calls it.

## Usage

```sh
offramp compose --kcc ~/src/k8s-config-connector --state terraform.tfstate --out charts/ --env prod
tofu show -json > state.json && offramp compose --kcc ... --state state.json --out charts/   # OpenTofu, encrypted or not
offramp compose ... --call gke_standard          # one call
offramp compose ... --single platform            # the whole state as one chart
```

The command exits 1 when any chart is not proven. `<out>/report.md`
summarizes the run, and each chart's `verify.md` gives its detail.

## Proved so far

`go test ./...` (`TestComposeFixtures`) composes every call of the four
synthetic fixture states and the live shared-vpc state: 22 charts, all
with zero diff. `TestVerifyCatchesDifference` checks that the check
fails on a changed default. `TestPlacementWiring` checks that overriding
`subnetwork` moves only the subnetwork ref.

## Next

- Read the root module's HCL, when it is given, so a module's own
  variables shape the values (the state alone cannot tell a hard-coded
  value from a variable).
- Let an agent propose further values (BND-61). The tier rules accept or
  reject them, and verify still decides.
- Wire `compose` into `offramp state` as the fallback before freezing.
- Run a composed chart live through the adoption path that `folders`
  proved, for a chart that has no catalog counterpart.
