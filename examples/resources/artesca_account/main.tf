resource "artesca_account" "team_a" {
  name  = "team-a"
  email = "team-a@example.com"
}

output "team_a_access_key" {
  value     = artesca_account.team_a.access_key
  sensitive = true
}

output "team_a_secret_key" {
  value     = artesca_account.team_a.secret_key
  sensitive = true
}

# Use the account credentials with the AWS provider for S3/IAM operations
provider "aws" {
  alias = "team_a"

  access_key = artesca_account.team_a.access_key
  secret_key = artesca_account.team_a.secret_key

  endpoints {
    s3  = "https://s3.artesca.example.com"
    iam = "https://iam.artesca.example.com"
    sts = "https://sts.artesca.example.com"
  }

  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  s3_use_path_style           = true
}

resource "aws_s3_bucket" "data" {
  provider = aws.team_a
  bucket   = "team-a-data"
}
