package provider

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/cycloidio/cycloid-cli/gen/models"
)

// TFPRO-78: the backend silently drops input variables that the stack form
// does not declare. Read would then return the configuration without them and
// the plan would show the same diff forever. The provider instead keeps the
// user's value in state for undeclared paths (so the plan converges) and emits
// a warning naming them. Declared variables keep the normal drift detection.

// errUseCaseNotFound reports a stack config without the component's use case.
type errUseCaseNotFound struct{ useCase string }

func (e errUseCaseNotFound) Error() string {
	return fmt.Sprintf("use case %q not found in stack config", e.useCase)
}

// declaredFormPaths returns the set of variable keys the stack form declares
// for useCase, over all its sections and groups. The backend identifies a
// variable by key alone: keys are unique per use case (svccat/form/forms.go),
// values are stored flattened (FormVariables.Flatten) and read back by key
// (svccat/form/forms_server.go), whatever section/group the config used.
func declaredFormPaths(configs models.ServiceCatalogConfigs, useCase string) (map[string]struct{}, error) {
	uc, ok := configs[useCase]
	if !ok {
		return nil, errUseCaseNotFound{useCase: useCase}
	}
	declared := make(map[string]struct{})
	if uc.Forms == nil {
		return declared, nil
	}
	for _, section := range uc.Forms.Sections {
		if section == nil {
			continue
		}
		for _, group := range section.Groups {
			if group == nil {
				continue
			}
			for _, v := range group.Vars {
				if v == nil {
					continue
				}
				declared[v.Key] = struct{}{}
			}
		}
	}
	return declared, nil
}

// undeclaredVariables returns the subset of vars whose key the form does not
// declare (in any section/group),
// and their dotted paths sorted for stable diagnostics.
func undeclaredVariables(vars map[string]map[string]map[string]any, declared map[string]struct{}) (map[string]map[string]map[string]any, []string) {
	out := make(map[string]map[string]map[string]any)
	var paths []string
	for sec, groups := range vars {
		for grp, keys := range groups {
			for k, v := range keys {
				if _, ok := declared[k]; ok {
					continue
				}
				if out[sec] == nil {
					out[sec] = make(map[string]map[string]any)
				}
				if out[sec][grp] == nil {
					out[sec][grp] = make(map[string]any)
				}
				out[sec][grp][k] = v
				paths = append(paths, fmt.Sprintf("input_variables.%s.%s.%s", sec, grp, k))
			}
		}
	}
	sort.Strings(paths)
	return out, paths
}

// overlayVariables writes every value of overlay into base (a copy is returned).
func overlayVariables(base, overlay map[string]map[string]map[string]any) map[string]map[string]map[string]any {
	out := cloneNestedStringMapAny(base)
	for sec, groups := range overlay {
		if out[sec] == nil {
			out[sec] = make(map[string]map[string]any)
		}
		for grp, keys := range groups {
			if out[sec][grp] == nil {
				out[sec][grp] = make(map[string]any)
			}
			for k, v := range keys {
				out[sec][grp][k] = v
			}
		}
	}
	return out
}

func undeclaredVariablesWarning(paths []string, useCase string) diag.Diagnostic {
	return diag.NewAttributeWarningDiagnostic(
		path.Root("input_variables"),
		"Input variables not declared by the stack form",
		fmt.Sprintf("%s not declared by the stack form for use case %q and ignored: the backend drops them, so Terraform does not diff them. Remove them from the configuration or declare them in the stack's .forms.yml.",
			describePaths(paths), useCase),
	)
}

func describePaths(paths []string) string {
	if len(paths) == 1 {
		return paths[0] + " is"
	}
	return strings.Join(paths, ", ") + " are"
}

// droppedPaths reports whether some path of userInput is absent from
// returned, i.e. the backend did not give it back. Only then can the form
// lookup change anything, so it gates the extra API call.
func droppedPaths(userInput, returned map[string]map[string]map[string]any) bool {
	for sec, groups := range userInput {
		for grp, keys := range groups {
			for k := range keys {
				if _, ok := returned[sec][grp][k]; !ok {
					return true
				}
			}
		}
	}
	return false
}

// keepUndeclaredVariables re-adds to inputVariables the undeclared entries of
// userInput (the configured or prior-state value), and warns about them. The
// stack config is only fetched when the backend dropped some path of
// userInput. When it cannot be checked, it warns and returns inputVariables
// as is.
func (r *ComponentResource) keepUndeclaredVariables(org, project, env, canonical, useCase string, userInput, inputVariables map[string]map[string]map[string]any) (map[string]map[string]map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(userInput) == 0 || !droppedPaths(userInput, inputVariables) {
		return inputVariables, diags
	}
	configs, _, err := r.provider.Client.GetComponentStackConfig(org, project, env, canonical, useCase, "", "", "")
	if err != nil {
		diags.AddAttributeWarning(
			path.Root("input_variables"),
			"Could not check input variables against the stack form",
			fmt.Sprintf("Fetching the stack config of component %q failed, so variables the form does not declare cannot be ignored and may show a diff: %s", canonical, err),
		)
		return inputVariables, diags
	}
	declared, err := declaredFormPaths(configs, useCase)
	if err != nil {
		diags.AddAttributeWarning(
			path.Root("input_variables"),
			"Stack config has no matching use case",
			fmt.Sprintf("The stack config of component %q has no use case %q, so undeclared variables could not be checked and may show a diff.", canonical, useCase),
		)
		return inputVariables, diags
	}
	undeclared, paths := undeclaredVariables(userInput, declared)
	if len(paths) == 0 {
		return inputVariables, diags
	}
	diags.Append(undeclaredVariablesWarning(paths, useCase))
	return overlayVariables(inputVariables, undeclared), diags
}

// warnUndeclaredVariables only returns the diagnostics of
// keepUndeclaredVariables, for call sites where state already holds the
// configured value: configured is what was sent, returned what the backend
// gave back.
func (r *ComponentResource) warnUndeclaredVariables(org, project, env, canonical, useCase string, configured, returned map[string]map[string]map[string]any) diag.Diagnostics {
	_, diags := r.keepUndeclaredVariables(org, project, env, canonical, useCase, configured, returned)
	return diags
}
