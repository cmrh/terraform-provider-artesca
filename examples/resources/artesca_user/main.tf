resource "artesca_user" "example" {
  account_name = artesca_account.example.name
  username     = "app-service"
}
