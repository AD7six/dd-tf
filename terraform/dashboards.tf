resource "datadog_dashboard_json" "dashboard" {
  # Loop on each dashhboard indexed by id
  for_each = local.dashboards_by_id

  # Use the json file contents verbatim
  dashboard = file(each.value)
}
