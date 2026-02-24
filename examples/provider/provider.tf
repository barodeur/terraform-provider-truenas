terraform {
  required_providers {
    truenas = {
      source = "barodeur/truenas"
    }
  }
}

provider "truenas" {
  host     = "truenas.local"
  api_key  = var.truenas_api_key
  insecure = true
}

variable "truenas_api_key" {
  type      = string
  sensitive = true
}
