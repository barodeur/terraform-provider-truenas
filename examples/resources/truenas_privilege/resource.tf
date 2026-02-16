resource "truenas_group" "operators" {
  name = "operators"
}

resource "truenas_privilege" "readonly_ops" {
  name         = "readonly-ops"
  local_groups = [truenas_group.operators.gid]
  roles        = ["READONLY_ADMIN"]
  web_shell    = false
}
