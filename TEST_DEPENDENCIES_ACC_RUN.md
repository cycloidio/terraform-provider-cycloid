# Acceptance test run: missing dependencies and issues

Summary of a full acceptance test run (`TF_ACC=1 go test ./provider/... -v`) and what is missing or broken in the test environment / test code.

---

## Tests that **passed**

- **TestAccCatalogRepositoryResource** – PASS
- **TestAccProjectResource** – PASS
- **TestAccCredentialResource_*** (AWS, SSH, BasicAuth, Azure, AzureStorage, GCP, Elasticsearch, Swift, Custom) – all PASS
- **TestCyModelToData** (unit) – PASS

---

## Failures and what’s missing

### 1. Component tests (all 5)

**Error:** `Failed to list version for stack "test-org:web-app-stack" in org "cycloid"` → **The Service Catalog was not found**.

**Missing test dependency:** A **stack / service catalog** that exists in the test org and can be referenced by component tests.

- **Current:** Tests use a hardcoded `stack_ref = "test-org:web-app-stack"`.
- **Needed:** Either:
  - Add a real stack canonical (e.g. `org:stack-name`) to the test config and ensure that stack exists in the test org, or
  - Create a stack in test setup (if the API allows it) and pass its ref into the component config.

**Suggestion:** Add to `test_config.yaml` (and TestConfig struct) something like:

- `stack_ref` (or `component.stack_ref`) – e.g. `"my-org:my-stack"` – and ensure the test env has that stack.

---

### 2. Config repository

**Error:** `The Git Repository is invalid: failed to read repository. Invalid URL: https://github.com/hashicorp/terraform`.

**Missing / mismatch:** The URL from test config (`https://github.com/hashicorp/terraform`) is either not cloneable by the backend, or the **credential** used (`credential_canonical: "github"`) does not have access to it.

**Needed:**

- A **config repository URL** that the test env can actually clone (with the configured credential).
- Or a **credential** in the test org that can access the URL in the config (and set `CY_TEST_CREDENTIAL_CANONICAL` / config to match).

---

### 3. Organization member

**Error:** `The Role was not found` (inviteUserToOrgMemberNotFound).

**Missing test dependency:** The **role** `"member"` (used in the test) does not exist in the test org.

**Needed:**

- Either create/use a role whose canonical exists in the org (e.g. a role that the API returns for that org), or
- Add a configurable **role_canonical** to test config and set it to a role that exists in the test environment.

---

### 4. Organization resource (WithAllowDestroy)

**Error:** `fail to read current organizations` → **The Organization was not found** (getChildrenNotFound).

**Missing / scope:** The test creates organizations (e.g. `test-org-destroy`). The API user (CY_API_KEY) may not have permission to create/list child organizations, or the endpoint used assumes an existing parent org.

**Needed:** Confirm the test API user can create and read organizations in the test env; or skip / restrict this test when the account cannot manage orgs.

---

### 5. Stack resource

**Error:** Terraform plan error: `A reference to a data source must be followed by at least one attribute access, specifying the resource name.`  
Config uses: `data.cycloid_stacks[0].existing.canonical`.

**Issue:** **Test code bug** – invalid reference. The data source is named `existing`, so the reference must use that name and the actual attribute path of the `cycloid_stacks` data source (e.g. `data.cycloid_stacks.existing.<attribute>` and then an index if it’s a list). It should not be `data.cycloid_stacks[0].existing.canonical`.

**Needed:** Fix the stack test Terraform config to use the correct `cycloid_stacks` data source schema (resource name + attribute path). No new test env dependency; just fix the HCL.

---

### 6. Team / Team member

**Errors:**

- `A Team with the same canonical: "test-team" already exists` (409) – name collision between tests or leftover from a previous run.
- `Resource Import Not Implemented` for `cycloid_team` – import step is not supported by the provider.

**Missing / issues:**

- **Test isolation:** Use unique team canonicals per test run (e.g. random suffix or test name) so parallel or repeated runs don’t conflict.
- **Import step:** Remove or skip the import step for `cycloid_team` until the provider supports import; otherwise the test will keep failing.

---

### 7. Environment

**Error:** `The Project was not found` (getProjectEnvironmentsNotFound / deleteEnvironmentNotFound). The test creates a project via `EnsureTestProject` then creates an environment; the project is not found when the environment is updated or destroyed.

**Possible causes:**

- Project is created in one org and the environment is created in another (e.g. org canonical mismatch).
- Timing or backend caching: project not visible yet.
- Test env or API restrictions on project creation.

**Needed:** Ensure project creation and environment creation use the same org and that the test env allows both; optionally add a short retry or ensure project canonical is used consistently.

---

## Recommended next steps

1. **Test config (YAML + env):**
   - Add **stack_ref** (for component tests) and **role_canonical** (for organization_member) to `test_config.yaml` and `TestConfig`, and document that the test env must have that stack and role.
2. **Config repository:** Set **repositories.config.url** (and credential) to a URL the test env can clone with the configured credential.
3. **Stack resource test:** Fix the data source reference in `testAccStackConfig_basic` / `_updated` so it matches the real `cycloid_stacks` schema.
4. **Team tests:** Use unique team names (e.g. from `t.Name()` or a random suffix) and remove or skip the import step for `cycloid_team` until import is implemented.
5. **Organization / Environment:** Align with API permissions and org/project visibility; add docs or skip conditions if the test account cannot create orgs or certain resources.

Once these are in place, re-run acceptance tests and adjust any remaining failures against this list.
