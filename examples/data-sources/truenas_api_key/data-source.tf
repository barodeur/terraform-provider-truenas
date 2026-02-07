data "truenas_api_key" "example" {
  name = "terraform-managed-key"
}

output "api_key_username" {
  value = data.truenas_api_key.example.username
}
