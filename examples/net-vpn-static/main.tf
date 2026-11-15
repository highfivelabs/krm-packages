# A net-vpn-static module call as a team has it today, after the example in
# modules/net-vpn-static/README.md of Cloud Foundation Fabric (Apache-2.0,
# Copyright Google LLC), pinned the way a team pins it. Its tunnels are left
# out: a tunnel's shared secret is a Terraform string, and the chart takes
# it only as a reference to a Kubernetes Secret (docs/secrets.md).

variable "project_id" {
  type = string
}

module "vpn" {
  source        = "github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/net-vpn-static?ref=v60.0.0"
  project_id    = var.project_id
  region        = "europe-west1"
  network       = "projects/my-project/global/networks/my-network"
  name          = "remote"
  remote_ranges = ["10.10.0.0/24"]
}
