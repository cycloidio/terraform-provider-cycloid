package provider

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNotFoundError(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "external backend 404 error",
			err:      errors.New("A 404 error was returned on \"getExternalBackend\" call with message: The External Backend was not found"),
			expected: true,
		},
		{
			name:     "catalog repository 404 error",
			err:      errors.New("A 404 error was returned on \"getServiceCatalogSource\" call with message: The Service Catalog Source was not found"),
			expected: true,
		},
		{
			name:     "config repository 404 error",
			err:      errors.New("A 404 error was returned on \"getConfigRepository\" call with message: The Config Repository was not found"),
			expected: true,
		},
		{
			name:     "generic notfound operation",
			err:      errors.New("A 404 error was returned on \"getRoleNotFound\" call"),
			expected: true,
		},
		{
			name:     "non not found error",
			err:      errors.New("A 500 error was returned on \"getConfigRepository\" call"),
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, isNotFoundError(testCase.err))
		})
	}
}
