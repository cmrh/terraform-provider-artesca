resource "artesca_bucket" "example" {
  name                = "my-data-bucket"
  location_constraint = artesca_location.ring.name
  versioning_enabled  = true
  account_name        = artesca_account.example.name
}
