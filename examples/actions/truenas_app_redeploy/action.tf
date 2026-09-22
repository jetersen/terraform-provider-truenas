# Redeploy an app (Terraform 1.14+).
action "truenas_app_redeploy" "example" {
  config {
    app_name = "nextcloud"
  }
}
