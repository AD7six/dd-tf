variable "dd_api_key" {
  description = "Datadog API key"
  type        = string
  sensitive   = true
}

variable "dd_app_key" {
  description = "Datadog application key"
  type        = string
  sensitive   = true
}

variable "dd_api_url" {
  description = "Datadog API URL (e.g., https://api.datadoghq.com, https://api.datadoghq.eu)"
  type        = string
  default     = "https://api.datadoghq.com"
}
