resource "truenas_api_key" "example" {
  name = "terraform-managed-key"
}

output "api_key_id" {
  value = truenas_api_key.example.id
}

output "api_key_value" {
  value     = truenas_api_key.example.key
  sensitive = true
}
