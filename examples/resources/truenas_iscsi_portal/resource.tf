resource "truenas_iscsi_portal" "example" {
  listen  = [{ ip = "0.0.0.0" }]
  comment = "Default portal"
}
