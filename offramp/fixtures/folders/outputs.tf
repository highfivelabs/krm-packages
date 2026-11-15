# Copyright 2026 High Five Labs
# SPDX-License-Identifier: Apache-2.0

output "root" {
  value = module.root.id
}

output "environments" {
  value = module.environments.ids
}

output "teams" {
  value = module.teams.ids
}
