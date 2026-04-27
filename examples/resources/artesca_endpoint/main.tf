resource "artesca_endpoint" "example" {
  hostname      = "data.s3.my-company.com"
  location_name = artesca_location.ring.name
}
