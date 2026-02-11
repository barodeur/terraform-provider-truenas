resource "truenas_cronjob" "example" {
  command     = "/usr/local/bin/backup.sh"
  user        = "root"
  description = "Daily backup at 2:30 AM"

  schedule = {
    minute = "30"
    hour   = "2"
    dom    = "*"
    month  = "*"
    dow    = "*"
  }
}
