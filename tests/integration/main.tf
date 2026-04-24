terraform {
  required_providers {
    artesca = {
      source = "scality/artesca"
    }
  }
}

provider "artesca" {}

# --- Account ---

resource "artesca_account" "test" {
  name  = "inttest-account"
  email = "inttest@example.com"
}

output "account_name" {
  value = artesca_account.test.name
}

output "account_arn" {
  value = artesca_account.test.arn
}

output "account_id" {
  value = artesca_account.test.id
}

output "account_canonical_id" {
  value = artesca_account.test.canonical_id
}

# --- IAM User ---

resource "artesca_user" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = "inttest-user"
}

output "user_username" {
  value = artesca_user.test.username
}

output "user_id" {
  value = artesca_user.test.user_id
}

output "user_arn" {
  value = artesca_user.test.arn
}

output "user_path" {
  value = artesca_user.test.path
}

# --- User Access Key ---

resource "artesca_user_access_key" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
}

output "user_access_key_id" {
  value = artesca_user_access_key.test.access_key_id
}

output "user_access_key_secret" {
  value     = artesca_user_access_key.test.secret_access_key
  sensitive = true
}

output "user_access_key_status" {
  value = artesca_user_access_key.test.status
}

# --- User Policy ---

resource "artesca_user_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
  policy_name        = "inttest-policy"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = "arn:aws:s3:::*"
      }
    ]
  })
}

output "user_policy_name" {
  value = artesca_user_policy.test.policy_name
}

# --- Location ---

variable "ring_s3_endpoint" {
  type      = string
  sensitive = false
}

variable "ring_s3_access_key" {
  type      = string
  sensitive = true
}

variable "ring_s3_secret_key" {
  type      = string
  sensitive = true
}

variable "ring_s3_bucket_name" {
  type = string
}

resource "artesca_location" "test" {
  name          = "inttest-ring-loc"
  location_type = "location-scality-ring-s3-v1"

  depends_on = [artesca_account.test]

  details {
    endpoint               = var.ring_s3_endpoint
    access_key             = var.ring_s3_access_key
    secret_key             = var.ring_s3_secret_key
    bucket_name            = var.ring_s3_bucket_name
    bucket_match           = false
    server_side_encryption = true
  }
}

output "location_name" {
  value = artesca_location.test.name
}

output "location_type" {
  value = artesca_location.test.location_type
}

output "location_object_id" {
  value = artesca_location.test.object_id
}

output "location_is_builtin" {
  value = artesca_location.test.is_builtin
}

# --- S3 Bucket ---

resource "artesca_bucket" "test" {
  name                = "inttest-bucket"
  location_constraint = artesca_location.test.name
  account_access_key  = artesca_account.test.access_key
  account_secret_key  = artesca_account.test.secret_key
}

output "bucket_name" {
  value = artesca_bucket.test.name
}

output "bucket_location" {
  value = artesca_bucket.test.location_constraint
}

# --- Destination Location ---

variable "dest_ring_s3_endpoint" {
  type      = string
  sensitive = false
}

variable "dest_ring_s3_access_key" {
  type      = string
  sensitive = true
}

variable "dest_ring_s3_secret_key" {
  type      = string
  sensitive = true
}

variable "dest_ring_s3_bucket_name" {
  type = string
}

resource "artesca_location" "dest" {
  name          = "inttest-ring-dest"
  location_type = "location-scality-ring-s3-v1"

  # Serialize location creation to avoid lost-update on the management overlay.
  depends_on = [artesca_location.test]

  details {
    endpoint               = var.dest_ring_s3_endpoint
    access_key             = var.dest_ring_s3_access_key
    secret_key             = var.dest_ring_s3_secret_key
    bucket_name            = var.dest_ring_s3_bucket_name
    bucket_match           = false
    server_side_encryption = true
  }
}

output "dest_location_name" {
  value = artesca_location.dest.name
}

output "dest_location_object_id" {
  value = artesca_location.dest.object_id
}

# --- Destination Bucket ---

resource "artesca_bucket" "dest" {
  name                = "inttest-bucket-dest"
  location_constraint = artesca_location.dest.name
  account_access_key  = artesca_account.test.access_key
  account_secret_key  = artesca_account.test.secret_key
}

output "dest_bucket_name" {
  value = artesca_bucket.dest.name
}

# --- Endpoint ---

resource "artesca_endpoint" "test" {
  hostname      = "inttest-bucket.s3.my-company.com"
  location_name = artesca_location.test.name

  depends_on = [artesca_account.test]
}

output "endpoint_hostname" {
  value = artesca_endpoint.test.hostname
}

# --- Workflow: Expiration ---

resource "artesca_bucket_workflow_expiration" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.test.name
  enabled            = true

  current_version_trigger_delay_days = 30

  filter {
    object_key_prefix = "logs/"
  }
}

output "expiration_rule_id" {
  value = artesca_bucket_workflow_expiration.test.rule_id
}

# --- Workflow: Transition ---

resource "artesca_bucket_workflow_transition" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.test.name
  enabled            = true
  location_name      = artesca_location.dest.name
  trigger_delay_days = 60

  # Both lifecycle rules modify the same S3 lifecycle configuration;
  # serialize to avoid a read-modify-write race.
  depends_on = [artesca_bucket_workflow_expiration.test]

  filter {
    object_key_prefix = "archive/"
  }
}

output "transition_rule_id" {
  value = artesca_bucket_workflow_transition.test.rule_id
}

# --- Replication ---

resource "artesca_replication" "test" {
  name    = "inttest-replication"
  version = 1
  enabled = true

  source {
    bucket_name = artesca_bucket.test.name
    prefix      = ""
    location    = artesca_location.test.name
  }

  destination {
    bucket_name = artesca_bucket.dest.name
    location    = artesca_location.dest.name

    locations {
      name = artesca_location.dest.name
    }
  }
}

output "replication_stream_id" {
  value = artesca_replication.test.stream_id
}
