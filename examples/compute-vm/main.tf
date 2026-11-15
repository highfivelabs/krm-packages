# A compute-vm module call as a team has it today, after the examples in
# modules/compute-vm/README.md of Cloud Foundation Fabric (Apache-2.0,
# Copyright Google LLC), pinned the way a team pins it.

variable "project_id" {
  type = string
}

module "web-vm" {
  source     = "github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/compute-vm?ref=v60.0.0"
  project_id = var.project_id
  zone       = "europe-west1-b"
  name       = "web-0"
  network_interfaces = [{
    network    = "projects/myproject/global/networks/vpc"
    subnetwork = "projects/myproject/regions/europe-west1/subnetworks/apps"
  }]
  attached_disks = {
    data = {
      initialize_params = {
        size = 20
        type = "pd-ssd"
      }
      snapshot_schedule = ["daily"]
    }
  }
  snapshot_schedules = {
    daily = {
      schedule = {
        daily = { days_in_cycle = 1, start_time = "03:00" }
      }
      retention_policy = { max_retention_days = 7 }
    }
  }
  service_account = {
    auto_create = true
  }
  tags   = ["http-server", "ssh"]
  labels = { app = "web" }
}
