# A container-registry module call as a team has it today, after the
# example in modules/container-registry/README.md of Cloud Foundation Fabric
# (Apache-2.0, Copyright Google LLC), pinned the way a team pins it. Its
# iam grants an Artifact Registry role: the registry is now an Artifact
# Registry gcr.io repository, which takes no Cloud Storage role
# (charts/gcp/container-registry/README.md).

module "container_registry" {
  source     = "github.com/GoogleCloudPlatform/cloud-foundation-fabric//modules/container-registry?ref=v60.0.0"
  project_id = "my-project"
  location   = "EU"
  iam = {
    "roles/artifactregistry.reader" = ["user:me@example.com"]
  }
}
