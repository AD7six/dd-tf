locals {
  # Root path to the dashboards JSON files exported by dd-tf. Update this to
  # match your needs
  dashboards_root = "${path.module}/../data/dashboards"

  # All JSON files recursively under dashboards_root
  dashboard_files = fileset(local.dashboards_root, "**/*.json")

  # Map dashboards by intrinsic id parsed from the JSON content for stable
  # for_each keys.
  dashboards_by_id = {
    for rel in local.dashboard_files :
    jsondecode(file("${local.dashboards_root}/${rel}")).id => "${local.dashboards_root}/${rel}"
  }
}
