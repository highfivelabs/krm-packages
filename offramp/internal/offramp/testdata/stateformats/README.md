# State forms from Terraform and OpenTofu

Each file was written by the tool its name gives, except `v3.tfstate`.
They test that `ReadState` reads every form both tools produce and refuses
the rest with what to run instead (`stateformats_test.go`).

| File | Written by | How |
| --- | --- | --- |
| `gke.terraform-1.17.2.show.json` | Terraform 1.17.2 | `terraform show -json` of `../synthetic/gke/terraform.tfstate` |
| `gke.opentofu.tfstate` | (edit) | `../synthetic/gke/terraform.tfstate` with its providers on `registry.opentofu.org`, as an OpenTofu-native state (or one after `tofu state replace-provider`) records them |
| `gke.opentofu-1.10.7.show.json` | OpenTofu 1.10.7 | `tofu show -json gke.opentofu.tfstate` |
| `gke.deposed.tfstate` | (edit) | the gke state with a deposed copy of `pools["apps"]`, as `create_before_destroy` leaves one when the old object's destroy fails; both tools read it |
| `gke.deposed.terraform-1.17.2.show.json` | Terraform 1.17.2 | `terraform show -json gke.deposed.tfstate` (`deposed_key`) |
| `opentofu-1.10.7.encrypted.tfstate` | OpenTofu 1.10.7 | `tofu apply` of one `terraform_data` with state encryption (pbkdf2, aes_gcm) |
| `sensitive.terraform-1.17.2.tfstate`, `sensitive.opentofu-1.10.7.tfstate` | each tool | `apply` of one `terraform_data` whose input holds a `sensitive()` value |
| `v3.tfstate` | (by hand) | the shape of a Terraform 0.11 state, format version 3 |
