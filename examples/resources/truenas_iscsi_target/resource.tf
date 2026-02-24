resource "truenas_iscsi_target" "example" {
  name   = "data-target"
  groups = [{
    portal = truenas_iscsi_portal.example.id
  }]
}
