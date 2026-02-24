resource "truenas_iscsi_extent" "example" {
  name     = "data-lun"
  type     = "FILE"
  path     = "/mnt/tank/iscsi/data-lun"
  filesize = 10737418240 # 10 GiB
}
