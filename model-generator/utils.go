/*
 * Copyright 2018, Automatic Inc.
 * All rights reserved.
 *
 * Author: Aniruddha Maru
 */

package main

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema"
)

// name creates the struct name for a given Hedwig message
func name(id string, majorVersion int) string {
	base := strcase.ToCamel(strings.Replace(id, ".", "_", -1))
	return fmt.Sprintf("%sV%d", base, majorVersion)
}

// toCamel converts a go name for struct, variable, field, ... to appropriate camel case
func toCamel(s string) string {
	camel := strcase.ToCamel(s)
	for initialism := range commonInitialisms {
		initialismTitle := strings.Title(strings.ToLower(initialism))
		if strings.HasSuffix(camel, initialismTitle) {
			camel = strings.TrimSuffix(camel, initialismTitle) + initialism
			// can't have any other initialism if one is found.
			break
		}
	}
	return camel
}

// getJsType gets the JSON Schema type given a list of types defined in the schema
func getJsType(types []string) string {
	jsType := ""
	if len(types) == 1 {
		return types[0]
	} else if len(types) == 2 {
		if types[0] == jsTypeNull {
			jsType = types[1]
		} else if types[1] == jsTypeNull {
			jsType = types[0]
		}
	}
	return jsType
}

// contains finds a value in a list of values
func contains(value string, values []string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

// nullAllowed determines if a given JSON Schema type allows null values
func nullAllowed(sch *jsonschema.Schema) bool {
	types := sch.Types
	if sch.Ref != nil {
		types = sch.Ref.Types
	}
	return contains(jsTypeNull, types)
}

// isRequired determines if a given property is required per JSON Schema
func isRequired(prop string, sch *jsonschema.Schema) bool {
	return contains(prop, sch.Required)
}

// getGoType determines the go type for a JSON Schema property
func getGoType(goTypes map[*jsonschema.Schema]string, nullable bool, sch *jsonschema.Schema) (string, error) {
	if sch.Ref != nil {
		sch = sch.Ref
	}

	jsType := getJsType(sch.Types)

	if jsType == "" {
		return "", errors.Errorf("unable to determine type for schema: %v", sch)
	}
	concreteType := ""
	typeNaturallyNullable := false
	switch jsType {
	case jsTypeNull:
		return "", errors.Errorf("unable to determine type for schema: %v", sch)
	case "integer":
		concreteType = "int"
	case "string":
		concreteType = "string"
	case "object":
		goType, ok := goTypes[sch]
		if !ok {
			return "", errors.Errorf("unable to determine type for schema: %v", sch)
		}
		concreteType = goType
		typeNaturallyNullable = true
	case "array":
		if sch.Items == nil {
			return "[]interface{}", nil
		}
		if itemsSchema, ok := sch.Items.(*jsonschema.Schema); ok {
			childType, err := getGoType(goTypes, false, itemsSchema)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("[]%s", childType), nil
		} else {
			return "[]interface{}", nil
		}
	case "boolean":
		concreteType = "bool"
	case "number":
		concreteType = "float64"
	default:
		return "", errors.Errorf("unknown primitive type: %v", jsType)
	}
	if nullable && !typeNaturallyNullable {
		return fmt.Sprintf("*%s", concreteType), nil
	}
	return concreteType, nil
}

// keys gets a list of map keys
func keys(fields map[string]*jsonschema.Schema) []string {
	keys := make([]string, len(fields))
	i := 0
	for k := range fields {
		keys[i] = k
		i++
	}
	return keys
}

// keysGeneric gets a list of map keys for a generic map
func keysGeneric(fields map[string]interface{}) []string {
	keys := make([]string, len(fields))
	i := 0
	for k := range fields {
		keys[i] = k
		i++
	}
	return keys
}

// msgKey struct represents a Hedwig message to be used in maps
func msgKey(msgType, version string) string {
	return fmt.Sprintf("%s.%s", msgType, version)
}
