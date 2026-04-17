resource "artesca_bucket_workflow_replication" "example" {
  account_id  = artesca_account.example.canonical_id
  bucket_name = "my-source-bucket"
  name        = "replicate-to-backup"
  version     = 1
  enabled     = true

  source {
    bucket_name = "my-source-bucket"
    prefix      = ""
  }

  destination {
    locations {
      name = artesca_location.backup.name
    }
  }
}
