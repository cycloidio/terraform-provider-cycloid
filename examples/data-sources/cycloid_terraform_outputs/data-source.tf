# Filter the output with key `some_cool_output` from the env `my-env` in project `my-project`
data "cycloid_terraform_outputs" "some_cool_output" {
  filters = [
    {
      attribute = "project_canonical"
      condition = "eq"
      value     = "my-project"
    },
    {
      attribute = "environment_canonical"
      condition = "eq"
      value     = "my-env"
    },
    {
      attribute = "output_key"
      condition = "eq"
      value     = "some_cool_output"
    }
  ]
}
