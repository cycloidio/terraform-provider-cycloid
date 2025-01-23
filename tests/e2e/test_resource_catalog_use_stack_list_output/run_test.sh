export TF_TEST_STEPS="setup pre_plan_apply apply plan_destroy destroy"

for step in $TF_TEST_STEPS; do
  run_step "$step"
done
