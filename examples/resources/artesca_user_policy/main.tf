resource "artesca_user_policy" "example" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  username           = artesca_user.example.username
  policy_name        = "s3-read-write"

  policy_document = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"]
        Resource = "arn:aws:s3:::my-data-bucket/*"
      }
    ]
  })
}
