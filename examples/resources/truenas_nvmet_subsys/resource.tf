resource "truenas_nvmet_subsys" "storage" {
  name           = "storage-subsys"
  allow_any_host = false
}
