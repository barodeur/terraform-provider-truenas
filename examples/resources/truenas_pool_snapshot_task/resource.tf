resource "truenas_pool_snapshot_task" "daily" {
  dataset        = "tank/data"
  recursive      = true
  lifetime_value = 30
  lifetime_unit  = "DAY"
  naming_schema  = "auto-%Y-%m-%d_%H-%M"

  schedule = {
    minute = "00"
    hour   = "0"
    dom    = "*"
    month  = "*"
    dow    = "*"
  }
}
