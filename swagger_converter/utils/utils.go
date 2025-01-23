package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"

	"dario.cat/mergo"
	"github.com/pkg/errors"
)

// RunCmd is an exec wrapper that will output combined logs to os stdout and stderr
func RunCmd(cmd string, args ...string) error {
	proc := exec.Command(cmd, args...)
	proc.Env = os.Environ()
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc.Run()
}

// CurlToFile will fetch file at url and wite it to a file named filename
func CurlToFile(url, filename string) error {
	fmt.Println("fetching data from", url)

	response, err := http.Get(url)
	if err != nil {
		return errors.Wrapf(err, "failed to fetch content from '%s'", url)
	}
	defer response.Body.Close()

	outputFile, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "failed to write content from '%s' to '%s'", url, filename)
	}

	fmt.Println("writing to", filename)
	_, err = io.Copy(outputFile, response.Body)
	return errors.Wrapf(err, "failed to write content from '%s' to '%s'", url, filename)
}

// DeleteNestedKey will remove a key inside a m map[string]interface{}
// The last elemen in path will be the deleted key
func DeleteNestedKey(m map[string]interface{}, path []string) (map[string]interface{}, error) {
	if len(path) == 1 {
		delete(m, path[0])
		return m, nil
	}

	key := path[0]
	if reflect.ValueOf(m[key]).Kind() != reflect.Map {
		return nil, errors.Errorf("Nested key '%s' in path '%v' is not a map, value: %v", key, path, reflect.ValueOf(m[key]))
	}

	nestedMap, ok := m[key].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("failed to get key '%s' in map '%v'", key, m)
	}

	finalMap, err := DeleteNestedKey(nestedMap, path[1:])
	if err != nil {
		return nil, err
	}

	return finalMap, err
}

// RenameNestedKey will rename a key
// the last element in path is the key to rename with newName
func RenameNestedKey(m map[string]interface{}, path []string, newName string) error {
	key := path[0]
	if len(path) == 1 {
		value := m[key]
		delete(m, key)
		m[newName] = value
		return nil
	}

	if reflect.ValueOf(m[key]).Kind() != reflect.Map {
		return errors.Errorf("Nested key '%s' in path '%v' is not a map, value: %v", key, path, reflect.ValueOf(m[key]))
	}

	nestedMap, ok := m[key].(map[string]interface{})
	if !ok {
		return errors.Errorf("failed to get key '%s' in map '%v'", key, m)
	}

	err := RenameNestedKey(nestedMap, path[1:], newName)
	if err != nil {
		return err
	}

	return nil
}

// UpdateMapValue will change a nested map value in a map[string]interface{}
// Can't traverse array
func UpdateMapValue(m map[string]interface{}, path []string, value interface{}) error {
	key := path[0]
	if len(path) == 1 {
		m[key] = value
		return nil
	}

	if reflect.ValueOf(m[key]).Kind() != reflect.Map {
		return errors.Errorf("Nested key '%s' in path '%v' is not a map, value: %v", key, path, reflect.ValueOf(m[key]))
	}

	nestedMap, ok := m[key].(map[string]interface{})
	if !ok {
		return errors.Errorf("failed to get key '%s' in map '%v'", key, m)
	}

	err := UpdateMapValue(nestedMap, path[1:], value)
	if err != nil {
		return err
	}

	return nil
}

// SwaggerMergeAllOf the properties of an allOf Keyword in a OpenAPI spec that is m located at path
// this will try to merge the propeties of the $ref and the other in the array
func SwaggerMergeAllOf(m map[string]interface{}, path []string) error {
	key := path[0]

	if len(path) == 1 {
		fmt.Printf("Trying to merge attributes of '%s'.\n", key)
		values, ok := m[key].(map[string]interface{})
		if !ok {
			return errors.Errorf("value of key '%s' is not a map.", key)
		}

		allOf, ok := values["allOf"].([]interface{})
		if !ok {
			return errors.Errorf("no allOf key was found in key '%s'", key)
		}

		var mergedProperties = values
		for _, properties := range allOf {
			var mappedProperties = properties.(map[string]interface{})
			if ref, ok := mappedProperties["$ref"].(string); ok {
				fmt.Printf("found ref '%s', merging properties\n", ref)
				expandedRef, err := SwaggerExpandRefProperties(m, ref)
				if err != nil {
					return err
				}

				err = mergo.MapWithOverwrite(&mergedProperties, expandedRef)
				if err != nil {
					return err
				}

				delete(mappedProperties, "$ref")
			}

			err := mergo.MapWithOverwrite(&mergedProperties, properties)
			if err != nil {
				return errors.Wrapf(err, "fail to merge properties of '%s'", path)
			}
		}

		// Delete allOf Keyword once we merged it
		delete(mergedProperties, "allOf")

		m[key] = mergedProperties
		return nil
	}

	if reflect.ValueOf(m[key]).Kind() != reflect.Map {
		return errors.Errorf("Nested key '%s' in path '%v' is not a map, value: %v", key, path, reflect.ValueOf(m[key]))
	}

	nestedMap, ok := m[key].(map[string]interface{})
	if !ok {
		return errors.Errorf("failed to get key '%s' in map '%v'", key, m)
	}

	return SwaggerMergeAllOf(nestedMap, path[1:])
}

// SwaggerExpandRefProperties will expand the properties of an OpenAPI $ref keyword in m
func SwaggerExpandRefProperties(m map[string]interface{}, ref string) (map[string]interface{}, error) {
	refKeys := strings.Split(strings.TrimLeft(ref, "#/"), "/")

	// Okay, I will be cheating here for now
	// This function is used in the context of SwaggerMergeAllOf
	// when we are in the 'components.schemas' key of the openAPI spec
	// since we don't have acces to the full openAPI object here
	// I consider that we are at this level

	// It's dirty but it's a script, If you need to change this, this means that requirements have change -.o.-

	// I can at least assert that
	if len(refKeys) != 3 || !(slices.Contains(refKeys, "components") && slices.Contains(refKeys, "schemas")) {
		return nil, errors.Errorf(
			"Reference '%s' is not contains in the 'components.schemas' section of the OpenAPI file.\n%s",
			ref, "This script does not support ref outside this key.")
	}
	key := refKeys[2]
	nested, ok := m[key].(map[string]interface{})
	if !ok {
		return nil, errors.Errorf("key '%s' from ref '%s' was not found in OpenAPI file", key, ref)
	}

	return nested, nil
}
