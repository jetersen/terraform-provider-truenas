# Kick off a pool scrub (Terraform 1.14+).
action "truenas_scrub_run" "example" {
  config {
    pool_id = 1
  }
}
