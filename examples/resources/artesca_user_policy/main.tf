resource "artesca_user_policy" "example" {
  account_name = artesca_account.example.name
  username     = artesca_user.example.username
  policy_name  = "s3-read-write"

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
