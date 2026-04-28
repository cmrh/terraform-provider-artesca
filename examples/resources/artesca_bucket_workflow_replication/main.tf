# Per-bucket workflow replication: bucket-to-bucket only.
# For location-based or multi-backend replication, use artesca_replication instead.
resource "artesca_bucket_workflow_replication" "example" {
  account_id  = artesca_account.example.id
  bucket_name = artesca_bucket.source.name
  name        = "replicate-to-backup"
  version     = 1
  enabled     = true

  source {
    bucket_name = artesca_bucket.source.name
    prefix      = ""
  }

  destination {
    bucket_name = artesca_bucket.dest.name
  }
}
