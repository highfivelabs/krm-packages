# Copyright 2026 High Five Labs
# SPDX-License-Identifier: Apache-2.0
#
# Shape 1's three organization policies on the root folder (one boolean,
# one list, one with a condition). They are in their own file so a live run
# can leave them out: setting a folder's organization policies needs
# roles/orgpolicy.policyAdmin, which is granted only on an organization,
# and the live-test account does not hold it. hack/live.sh drops this file
# when ORG_POLICIES=false.

module "skip_default_network" {
  source  = "terraform-google-modules/org-policy/google//modules/org_policy_v2"
  version = "7.4.0"

  policy_root    = "folder"
  policy_root_id = trimprefix(module.root.id, "folders/")
  constraint     = "compute.skipDefaultNetworkCreation"
  policy_type    = "boolean"
  rules          = [{ enforcement = true }]
}

module "resource_locations" {
  source  = "terraform-google-modules/org-policy/google//modules/org_policy_v2"
  version = "7.4.0"

  policy_root    = "folder"
  policy_root_id = trimprefix(module.root.id, "folders/")
  constraint     = "gcp.resourceLocations"
  policy_type    = "list"
  rules          = [{ enforcement = true, allow = ["in:us-locations"] }]
}

module "require_os_login" {
  source  = "terraform-google-modules/org-policy/google//modules/org_policy_v2"
  version = "7.4.0"

  policy_root    = "folder"
  policy_root_id = trimprefix(module.root.id, "folders/")
  constraint     = "compute.requireOsLogin"
  policy_type    = "boolean"
  rules = [
    {
      enforcement = true
      conditions  = []
    },
    {
      enforcement = false
      conditions = [{
        title       = "break-glass"
        description = "Projects tagged break-glass may skip OS Login"
        expression  = "resource.matchTag('${var.org_id}/break-glass', 'true')"
        location    = ""
      }]
    },
  ]
}
