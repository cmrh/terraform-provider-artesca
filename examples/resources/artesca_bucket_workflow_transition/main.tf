resource "artesca_bucket_workflow_transition" "example" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  bucket_name        = artesca_bucket.example.name
  enabled            = true
  location_name      = artesca_location.archive.name
  trigger_delay_days = 30

  filter {
    object_key_prefix = "data/"
  }
}
