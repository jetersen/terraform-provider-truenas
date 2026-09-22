# Start an app (Terraform 1.14+).
action "truenas_app_start" "example" {
  config {
    app_name = "nextcloud"
  }
}
