# Azure fixtures

tf-infra's Azure units, as their states would look after an apply. Each
was made from the unit's read-only plan (`tofu show -json`, nothing
applied), with the values Azure fills in on apply added: ARM IDs from
parent, type and name, the identities' principal and client IDs, role
assignment GUIDs, and the alias's subscription ID.

| File | tf-infra unit | Modules |
| --- | --- | --- |
| `network.tfstate` | `terragrunt/dev/azure/networks/nobnt-cus-1` | `terraform/azure/network`: AVM resource group, virtual network, NAT gateway; the cluster identity and its role on the VNet |
| `cluster.tfstate` | `terragrunt/dev/azure/clusters/nobnt-cus-1` | `Azure/avm-res-containerservice-managedcluster` 0.8.8 (AKS Automatic) |
| `subscriptions.tfstate` | `terragrunt/global/azure/subscriptions` | `Azure/avm-ptn-alz` 0.23.0 and `Azure/avm-ptn-alz-sub-vending` 0.3.4, with the CI identities |

Every subscription, tenant, principal and billing ID is fake:

| ID | Stands for |
| --- | --- |
| `11111111-1111-1111-1111-111111111111` | hfl-bundt-dev (dev/azure) |
| `22222222-2222-2222-2222-222222222222` | hfl-us-bundt (prod/azure) |
| `33333333-3333-3333-3333-333333333333` | hfl-test (the adopted sandbox) |
| `44444444-4444-4444-4444-444444444444` | the tenant (its root management group) |

Principal, client and role assignment IDs are UUIDv5s of fixed strings.
The billing scope is all zeros. The federated credential's subject is a
fake 21-digit service account ID.
