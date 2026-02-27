resource "truenas_nvmet_port" "tcp" {
  addr_trtype  = "TCP"
  addr_traddr  = "0.0.0.0"
  addr_trsvcid = 4420
}
