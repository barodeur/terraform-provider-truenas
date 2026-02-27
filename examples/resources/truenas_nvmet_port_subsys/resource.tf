resource "truenas_nvmet_port_subsys" "bind" {
  port_id   = truenas_nvmet_port.tcp.id
  subsys_id = truenas_nvmet_subsys.storage.id
}
