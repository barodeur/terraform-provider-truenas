resource "truenas_pool_dataset" "shared" {
  name = "tank/shared"
}

resource "truenas_smb_share" "shared" {
  name    = "shared"
  path    = truenas_pool_dataset.shared.mountpoint
  comment = "Shared files"
}
