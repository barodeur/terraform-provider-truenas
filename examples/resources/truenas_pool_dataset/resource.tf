# Basic dataset
resource "truenas_pool_dataset" "data" {
  name = "tank/data"
}

# Dataset with options
resource "truenas_pool_dataset" "media" {
  name        = "tank/media"
  compression = "LZ4"
  atime       = "OFF"
  quota       = 1099511627776 # 1 TiB
  comments    = "Media storage managed by Terraform"
}

# Nested dataset with automatic parent creation
resource "truenas_pool_dataset" "nested" {
  name             = "tank/apps/postgres/data"
  create_ancestors = true
  recordsize       = "128K"
  sync             = "ALWAYS"
}
