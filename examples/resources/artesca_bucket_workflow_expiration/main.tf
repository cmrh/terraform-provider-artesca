resource "artesca_bucket_workflow_expiration" "example" {
  account_name = artesca_account.example.name
  bucket_name  = artesca_bucket.example.name
  enabled      = true

  current_version_trigger_delay_days = 90

  filter {
    object_key_prefix = "logs/"
  }
}
