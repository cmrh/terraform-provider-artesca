terraform {
  required_providers {
    artesca = {
      source = "registry.opentofu.org/cmrh/artesca"
    }
  }
}

provider "artesca" {
  management_endpoint = "https://management.artesca.example.com"
  oidc_url            = "https://ui.artesca.example.com"
  oidc_realm          = "artesca"
  client_id           = "zenko-ui"
  username            = var.artesca_username
  password            = var.artesca_password
  insecure_skip_verify = true

  # instance_id is auto-discovered if omitted
}

variable "artesca_username" {
  type      = string
  sensitive = false
}

variable "artesca_password" {
  type      = string
  sensitive = true
}
