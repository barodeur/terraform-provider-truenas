resource "truenas_group" "media" {
  name = "media"
  smb  = true
}

resource "truenas_user" "john" {
  username  = "john"
  full_name = "John Doe"
  group     = truenas_group.media.id
  smb       = true
  home      = "/mnt/tank/home/john"
  shell     = "/usr/bin/bash"
}
