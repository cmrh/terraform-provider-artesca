resource "artesca_bucket" "example" {
  name                = "my-data-bucket"
  location_constraint = artesca_location.ring.name
  versioning_enabled  = true
  account_access_key  = artesca_account.example.access_key
  account_secret_key  = artesca_account.example.secret_key
}
