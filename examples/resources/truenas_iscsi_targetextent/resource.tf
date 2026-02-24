resource "truenas_iscsi_targetextent" "example" {
  target = truenas_iscsi_target.example.id
  extent = truenas_iscsi_extent.example.id
}
