# Trigger an on-demand run of a cloud sync task (Terraform 1.14+).
action "truenas_cloudsync_run" "example" {
  config {
    id = 1
  }
}
