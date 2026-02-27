data "truenas_nvmet_global" "config" {
}

output "basenqn" {
  value = data.truenas_nvmet_global.config.basenqn
}
