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

output "account_id" {
  value = artesca_account.test.id
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

# --- User Access Key ---

resource "artesca_user_access_key" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
}

output "user_access_key_id" {
  value = artesca_user_access_key.test.access_key_id
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

# --- IAM Group + Inline Policy + Membership ---

resource "artesca_group" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = "inttest-group"
}

resource "artesca_group_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  group_name         = artesca_group.test.name
  policy_name        = "inttest-group-policy"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = "arn:aws:s3:::*"
      }
    ]
  })
}

resource "artesca_group_membership" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  group_name         = artesca_group.test.name
  username           = artesca_user.test.username
}

# --- IAM Managed Policy + Attachments (user/group/role) ---

resource "artesca_policy" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = "inttest-managed-policy"
  description        = "Integration test managed policy"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = "arn:aws:s3:::*"
      }
    ]
  })
}

resource "artesca_user_policy_attachment" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  username           = artesca_user.test.username
  policy_arn         = artesca_policy.test.arn
}

resource "artesca_group_policy_attachment" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  group_name         = artesca_group.test.name
  policy_arn         = artesca_policy.test.arn
}

# --- IAM Role + Attachment ---

resource "artesca_role" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  name               = "inttest-role"
  description        = "Integration test role"

  assume_role_policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "sts:AssumeRole"
      }
    ]
  })
}

resource "artesca_role_policy_attachment" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  role_name          = artesca_role.test.name
  policy_arn         = artesca_policy.test.arn
}

# --- STS AssumeRole (ephemeral) ---

ephemeral "artesca_assumed_role_credentials" "test" {
  access_key        = artesca_user_access_key.test.access_key_id
  secret_key        = artesca_user_access_key.test.secret_access_key
  role_arn          = artesca_role.test.arn
  role_session_name = "inttest-session"
  duration_seconds  = 3600

  # Ensure the attachment exists before we try to assume the role — otherwise
  # the session would succeed but have no permissions, which is fine for this
  # smoke test but ordering is cleaner this way.
  depends_on = [artesca_role_policy_attachment.test]
}

check "assume_role_returned_credentials" {
  assert {
    condition     = ephemeral.artesca_assumed_role_credentials.test.assumed_role_arn != ""
    error_message = "ephemeral assumed_role_arn was empty"
  }
  assert {
    condition     = ephemeral.artesca_assumed_role_credentials.test.expiration != ""
    error_message = "ephemeral expiration was empty"
  }
}

# --- Location (source) ---

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

resource "artesca_location" "source" {
  name          = "inttest-ring-loc"
  location_type = "location-scality-ring-s3-v1"

  details {
    endpoint               = var.ring_s3_endpoint
    access_key             = var.ring_s3_access_key
    secret_key             = var.ring_s3_secret_key
    bucket_name            = var.ring_s3_bucket_name
    bucket_match           = false
    server_side_encryption = true
  }
}

output "source_location_name" {
  value = artesca_location.source.name
}

# --- Location (destination) ---

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

# --- Source Bucket ---

resource "artesca_bucket" "source" {
  name                = "inttest-bucket"
  location_constraint = artesca_location.source.name
  versioning_enabled  = true
  account_access_key  = artesca_account.test.access_key
  account_secret_key  = artesca_account.test.secret_key
}

output "source_bucket_name" {
  value = artesca_bucket.source.name
}

# --- Destination Bucket ---

resource "artesca_bucket" "dest" {
  name                = "inttest-bucket-dest"
  location_constraint = artesca_location.dest.name
  versioning_enabled  = true
  account_access_key  = artesca_account.test.access_key
  account_secret_key  = artesca_account.test.secret_key
}

output "dest_bucket_name" {
  value = artesca_bucket.dest.name
}

# --- Bucket Policy ---

resource "artesca_bucket_policy" "source" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.source.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "AllowAccountRead"
      Effect = "Allow"
      # Workaround: artesca_account.test.arn returns a malformed IAM ARN
      # (arn:aws:iam::<id>:/<name>/) that the S3 policy validator rejects.
      # Use the canonical account-root principal until the server returns the
      # correct ARN format.
      Principal = { AWS = "arn:aws:iam::${artesca_account.test.id}:root" }
      Action    = ["s3:GetObject", "s3:ListBucket"]
      Resource = [
        "arn:aws:s3:::${artesca_bucket.source.name}",
        "arn:aws:s3:::${artesca_bucket.source.name}/*",
      ]
    }]
  })
}

# --- Bucket Tagging ---

resource "artesca_bucket_tagging" "source" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.source.name

  tags = {
    environment = "integration"
    managed_by  = "terraform"
  }
}

# --- Endpoint ---

resource "artesca_endpoint" "test" {
  hostname      = "inttest-bucket.s3.my-company.com"
  location_name = artesca_location.source.name
}

output "endpoint_hostname" {
  value = artesca_endpoint.test.hostname
}

# --- Workflow: Transition ---

resource "artesca_bucket_workflow_transition" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.source.name
  enabled            = true
  location_name      = artesca_location.dest.name
  trigger_delay_days = 60

  filter {
    object_key_prefix = "archive/"
  }
}

output "transition_rule_id" {
  value = artesca_bucket_workflow_transition.test.rule_id
}

# --- Workflow: Expiration ---

resource "artesca_bucket_workflow_expiration" "test" {
  account_access_key = artesca_account.test.access_key
  account_secret_key = artesca_account.test.secret_key
  bucket_name        = artesca_bucket.source.name
  enabled            = true

  current_version_trigger_delay_days = 30

  filter {
    object_key_prefix = "logs/"
  }
}

output "expiration_rule_id" {
  value = artesca_bucket_workflow_expiration.test.rule_id
}

# --- Bucket Workflow Replication ---

resource "artesca_bucket_workflow_replication" "test" {
  account_id  = artesca_account.test.id
  bucket_name = artesca_bucket.source.name
  name        = "inttest-wf-replication"
  version     = 1
  enabled     = true

  source {
    bucket_name = artesca_bucket.source.name
    prefix      = ""
    location    = artesca_location.source.name
  }

  destination {
    bucket_name = artesca_bucket.dest.name
  }
}

output "wf_replication_workflow_id" {
  value = artesca_bucket_workflow_replication.test.workflow_id
}
