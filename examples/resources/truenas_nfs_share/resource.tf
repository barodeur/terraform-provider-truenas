resource "truenas_pool_dataset" "exports" {
  name = "tank/exports"
}

resource "truenas_nfs_share" "exports" {
  path     = truenas_pool_dataset.exports.mountpoint
  comment  = "NFS exports"
  networks = ["192.168.1.0/24"]
}
