# Stop an app (Terraform 1.14+).
action "truenas_app_stop" "example" {
  config {
    app_name = "nextcloud"
  }
}
