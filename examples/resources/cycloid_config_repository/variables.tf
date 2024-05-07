variable "cycloid_api_key" {
    description = "Cycloid API key"
}

variable "cycloid_organization" {
    description = "Cycloid organization"
}

variable "cycloid_api_url" {
    description = "Cycloid API URL"
    default = "https://http-api.cycloid.io/"
}

variable "credential_ssh_key" {
    description = "SSH key to use for the credential"
}

variable "config_repository_url" {
    description = "URL of the repository containing the configuration"
}

variable "config_repository_branch" {
    description = "Branch of the repository containing the configuration"
    default = "main"
}
