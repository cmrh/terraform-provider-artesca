resource "artesca_user_access_key" "example" {
  account_access_key = artesca_account.example.access_key
  account_secret_key = artesca_account.example.secret_key
  username           = artesca_user.example.username
}

output "user_access_key_id" {
  value = artesca_user_access_key.example.access_key_id
}

output "user_secret_access_key" {
  value     = artesca_user_access_key.example.secret_access_key
  sensitive = true
}
