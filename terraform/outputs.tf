output "dashboard_ids" {
  description = "All dashboard IDs managed by this Terraform config"
  value       = keys(local.dashboards_by_id)
}
