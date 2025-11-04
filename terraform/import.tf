# After downloading dashboards - import them to add them to terraform state.
# Otherwise, copies of the dashboards will be created.
# Requires Terraform >= 1.5.0

# Create a block like this for each dashboard
#import {
#  to = datadog_dashboard_json.dashboard["abc-def-ghi"]
#  id = "abc-def-ghi"
#}
