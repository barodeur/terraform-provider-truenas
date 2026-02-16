resource "truenas_iscsi_auth" "example" {
  tag    = 1
  user   = "chapuser"
  secret = "mysecretpasswd"
}
