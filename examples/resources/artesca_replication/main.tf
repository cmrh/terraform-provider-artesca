# Overlay replication: location-based or multi-backend.
# For bucket-to-bucket replication, use artesca_bucket_workflow_replication instead.
resource "artesca_replication" "example" {
  name    = "replicate-to-archive"
  enabled = true

  source {
    bucket_name = artesca_bucket.source.name
    prefix      = ""
  }

  destination {
    locations {
      name = artesca_location.archive.name
    }
  }
}
