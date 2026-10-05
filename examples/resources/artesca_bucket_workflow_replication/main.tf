# Per-bucket replication rule: bucket-to-bucket.
# For location-based or multi-backend replication, use artesca_replication instead.
resource "artesca_bucket_workflow_replication" "example" {
  account_name            = artesca_account.example.name
  bucket_name             = artesca_bucket.source.name
  destination_bucket_name = artesca_bucket.dest.name
  enabled                 = true

  filter {
    object_key_prefix = "logs/"
  }
}
