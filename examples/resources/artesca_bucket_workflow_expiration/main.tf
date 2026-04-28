resource "artesca_bucket_workflow_expiration" "example" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  bucket_name        = artesca_bucket.example.name
  enabled            = true

  current_version_trigger_delay_days = 90

  filter {
    object_key_prefix = "logs/"
  }
}
