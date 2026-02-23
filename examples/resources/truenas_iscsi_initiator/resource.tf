resource "truenas_iscsi_initiator" "example" {
  initiators = ["iqn.2025-01.com.example:client1"]
  comment    = "Web server initiator"
}
