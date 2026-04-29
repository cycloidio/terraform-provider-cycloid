package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLhsEscapeValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain string", "dg-growy", "dg-growy"},
		{"dot wildcard", "dg.growy", "dg.growy"},
		{"question mark", "dg.?growy", "dg.?growy"},
		{"star wildcard", "dg.*growy", "dg.*growy"},
		{"plus quantifier", "dg.+growy", "dg.+growy"},
		{"brackets", "[A-Z]+", "[A-Z]+"},
		{"pipe alternation", "foo|bar", "foo|bar"},
		{"caret anchor", "^foo", "^foo"},
		{"dollar anchor", "foo$", "foo$"},
		{"backslash escape", `foo\.bar`, `foo\.bar`},
		{"parens groups", "(foo|bar)", "(foo|bar)"},
		{"curly braces", "a{2,4}", "a{2,4}"},
		{"ampersand encoded", "foo&bar", "foo%26bar"},
		{"equals encoded", "foo=bar", "foo%3Dbar"},
		{"hash encoded", "foo#bar", "foo%23bar"},
		{"space encoded", "foo bar", "foo%20bar"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lhsEscapeValue(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildLHSFilterQuery(t *testing.T) {
	tests := []struct {
		name    string
		filters []lhsFilter
		want    string
	}{
		{
			name:    "empty filters",
			filters: nil,
			want:    "",
		},
		{
			name: "single rlike filter",
			filters: []lhsFilter{
				{Attribute: "output_key", Condition: "rlike", Value: "dg.?growy"},
			},
			want: "output_key[rlike]=dg.?growy",
		},
		{
			name: "rlike with star",
			filters: []lhsFilter{
				{Attribute: "project_canonical", Condition: "rlike", Value: "dg.*growy"},
			},
			want: "project_canonical[rlike]=dg.*growy",
		},
		{
			name: "eq filter",
			filters: []lhsFilter{
				{Attribute: "output_key", Condition: "eq", Value: "my-key"},
			},
			want: "output_key[eq]=my-key",
		},
		{
			name: "multiple filters",
			filters: []lhsFilter{
				{Attribute: "output_key", Condition: "rlike", Value: "dg.?growy"},
				{Attribute: "project_canonical", Condition: "eq", Value: "my-project"},
			},
			want: "output_key[rlike]=dg.?growy&project_canonical[eq]=my-project",
		},
		{
			name: "value with ampersand",
			filters: []lhsFilter{
				{Attribute: "output_key", Condition: "eq", Value: "foo&bar"},
			},
			want: "output_key[eq]=foo%26bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildLHSFilterQuery(tt.filters)
			assert.Equal(t, tt.want, got)
		})
	}
}
