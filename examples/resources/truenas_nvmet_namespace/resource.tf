resource "truenas_nvmet_namespace" "vol" {
  subsys_id   = truenas_nvmet_subsys.storage.id
  device_type = "FILE"
  device_path = "/mnt/tank/nvme-ns0"
  filesize    = 10737418240
}
