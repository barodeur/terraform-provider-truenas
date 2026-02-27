resource "truenas_nvmet_host_subsys" "allow" {
  host_id   = truenas_nvmet_host.initiator.id
  subsys_id = truenas_nvmet_subsys.storage.id
}
