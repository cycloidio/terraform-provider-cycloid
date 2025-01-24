variable "cycloid_api_key" {
  description = "Cycloid API key - better fill it using TF_VAR_cycloid_api_key"
  sensitive   = true
  type        = string
}

variable "cycloid_organization" {
  description = "Cycloid organization"
  type        = string
}

variable "cycloid_api_url" {
  description = "Cycloid API URL"
  type        = string
  default     = "https://http-api.cycloid.io/"
}
