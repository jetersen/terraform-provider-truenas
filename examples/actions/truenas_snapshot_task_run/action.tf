# Trigger an on-demand run of a periodic snapshot task (Terraform 1.14+).
action "truenas_snapshot_task_run" "example" {
  config {
    id = 1
  }
}
