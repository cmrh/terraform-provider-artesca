resource "artesca_location" "aws_s3" {
  name          = "my-aws-location"
  location_type = "location-aws-s3-v1"

  details {
    access_key             = var.aws_access_key
    secret_key             = var.aws_secret_key
    bucket_name            = "my-target-bucket"
    bucket_match           = true
    region                 = "us-east-1"
    server_side_encryption = false
  }
}

resource "artesca_location" "ring_s3" {
  name          = "my-ring-location"
  location_type = "location-scality-ring-s3-v1"

  details {
    access_key  = var.ring_access_key
    secret_key  = var.ring_secret_key
    bucket_name = "ring-bucket"
    endpoint    = "https://ring.internal:8443"
  }
}
