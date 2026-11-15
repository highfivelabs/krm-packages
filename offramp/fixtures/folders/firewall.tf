# Copyright 2026 High Five Labs
# SPDX-License-Identifier: Apache-2.0
#
# Shape 1's hierarchical firewall policy, owned by the root folder and
# attached to prod and nonprod. In its own file so a live run can leave it
# out: Compute Engine names a hierarchical firewall policy's operations
# after the organization, so waiting on them needs
# compute.globalOperations.get on the organization, which the live-test
# account does not hold. hack/live.sh drops this file when
# FOLDER_FIREWALL=false.

module "hierarchical_firewall" {
  source  = "terraform-google-modules/network/google//modules/hierarchical-firewall-policy"
  version = "18.3.1"

  parent_node = module.root.id
  policy_name = "${var.prefix}-baseline"
  description = "offramp fixture: baseline rules for every project below"
  # Folder numbers: the module writes folders/<n> itself.
  target_folders = [
    trimprefix(module.environments.ids["prod"], "folders/"),
    trimprefix(module.environments.ids["nonprod"], "folders/"),
  ]

  rules = [
    {
      priority  = 1000
      direction = "INGRESS"
      action    = "allow"
      rule_name = "allow-iap"
      match = {
        src_ip_ranges  = ["35.235.240.0/20"]
        layer4_configs = [{ ip_protocol = "tcp", ports = ["22", "3389"] }]
      }
    },
    {
      priority  = 2000
      direction = "INGRESS"
      action    = "goto_next"
      rule_name = "delegate-internal"
      match = {
        src_ip_ranges  = ["10.0.0.0/8"]
        layer4_configs = [{ ip_protocol = "all" }]
      }
    },
    {
      priority  = 65000
      direction = "INGRESS"
      action    = "deny"
      rule_name = "deny-rest"
      match = {
        src_ip_ranges  = ["0.0.0.0/0"]
        layer4_configs = [{ ip_protocol = "all" }]
      }
    },
  ]
}

output "firewall_policy" {
  value = module.hierarchical_firewall.fw_policy.name
}
