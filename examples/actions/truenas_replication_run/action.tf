# Trigger an on-demand run of a replication task (Terraform 1.14+).
action "truenas_replication_run" "example" {
  config {
    id = 1
  }
}
