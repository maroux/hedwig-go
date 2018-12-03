/*
 * Copyright 2018, Automatic Inc.
 * All rights reserved.
 *
 * Author: Aniruddha Maru
 */

package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"io/ioutil"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema"
	"github.com/santhosh-tekuri/jsonschema/formats"
)

// fieldDocstring returns Go docstring for a JSON Schema property
func fieldDocstring(sch *jsonschema.Schema, indent, name string) string {
	if sch.Description == "" {
		return ""
	}
	cleaned := strings.Replace(strings.TrimSpace(sch.Description), "\n", "\n\t// ", -1)
	return fmt.Sprintf("%s// %s - %s\n", indent, name, cleaned)
}

// createGoStruct creates a Go struct to represent a JSON Schema object
func createGoStruct(
	sourceBuilder *strings.Builder, goTypes map[*jsonschema.Schema]string, name string, docstring string,
	sch *jsonschema.Schema) error {

	if docstring != "" {
		_, err := sourceBuilder.WriteString(docstring)
		if err != nil {
			return errors.Wrap(err, "unable to write")
		}
	}
	if sch.Properties == nil || len(sch.Properties) == 0 {
		_, err := sourceBuilder.WriteString(fmt.Sprintf("type %v map[string]interface{}\n\n", name))
		if err != nil {
			return errors.Wrap(err, "unable to write")
		}
	} else {
		_, err := sourceBuilder.WriteString(fmt.Sprintf("type %v struct {\n", name))
		if err != nil {
			return errors.Wrap(err, "unable to write")
		}
		propNames := keys(sch.Properties)
		sort.Strings(propNames)
		firstProp := true
		for _, name := range propNames {
			prop := sch.Properties[name]
			goType, err := getGoType(goTypes, nullAllowed(prop), prop)
			if err != nil {
				return err
			}
			fieldName := toCamel(name)
			docstring := fieldDocstring(prop, "\t", fieldName)
			if docstring != "" {
				if !firstProp {
					_, err := sourceBuilder.WriteRune('\n')
					if err != nil {
						return errors.Wrap(err, "unable to write")
					}
				}
				_, err := sourceBuilder.WriteString(docstring)
				if err != nil {
					return errors.Wrap(err, "unable to write")
				}
			}
			jsonTag := ""
			if !isRequired(name, sch) {
				jsonTag = ",omitempty"
			}
			typeDecl := fmt.Sprintf("\t%s  %s  `json:\"%s%s\"`\n", fieldName, goType, name, jsonTag)
			_, err = sourceBuilder.WriteString(typeDecl)
			if err != nil {
				return errors.Wrap(err, "unable to write")
			}
			firstProp = false
		}
		_, err = sourceBuilder.WriteString("}\n\n")
		if err != nil {
			return errors.Wrap(err, "unable to write")
		}
	}
	goTypes[sch] = name
	return nil
}

// createGoTypes creates all the types required to represent a given schema
func createGoTypes(
	sourceBuilder *strings.Builder, goTypes map[*jsonschema.Schema]string, names []string, sch *jsonschema.Schema,
) error {

	if sch.Ref != nil {
		if !strings.HasPrefix(sch.Ref.Ptr, "#/definitions/") {
			return errors.New("can't handle schema reference")
		}
		names = strings.Split(sch.Ref.Ptr[len("#/definitions/"):], "/")
	}

	if sch.Ref != nil {
		sch = sch.Ref
	}

	if _, ok := goTypes[sch]; ok {
		// type already exists
		return nil
	}

	jsType := getJsType(sch.Types)

	switch jsType {
	case "object":
		propertyNames := keys(sch.Properties)
		sort.Strings(propertyNames)
		for _, n := range propertyNames {
			namesCopy := make([]string, len(names)+1)
			copy(namesCopy, names)
			namesCopy[len(names)] = n
			err := createGoTypes(sourceBuilder, goTypes, namesCopy, sch.Properties[n])
			if err != nil {
				return err
			}
		}
		name := strcase.ToCamel(strings.Join(names, "_"))
		docstring := fieldDocstring(sch, "", name)
		err := createGoStruct(sourceBuilder, goTypes, name, docstring, sch)
		if err != nil {
			return err
		}
	case "array":
		if itemsSchema, ok := sch.Items.(*jsonschema.Schema); ok {
			names = append(names, "list")
			err := createGoTypes(sourceBuilder, goTypes, names, itemsSchema)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// createGoTypesForMessage creates all the types required to represent a given Hedwig message and all its child
// properties
func createGoTypesForMessage(
	sourceBuilder *strings.Builder, goTypes map[*jsonschema.Schema]string, msgType string, majorVersion int,
	msgSchema *jsonschema.Schema) (string, error) {

	msgStructName := name(msgType, majorVersion)
	if len(msgSchema.Types) != 0 && (len(msgSchema.Types) != 1 || msgSchema.Types[0] != "object") {
		return "", errors.Errorf("invalid msg schema with type: %v", msgSchema.Types)
	} else {
		propertyNames := keys(msgSchema.Properties)
		sort.Strings(propertyNames)
		for _, n := range propertyNames {
			err := createGoTypes(sourceBuilder, goTypes, []string{msgStructName, n}, msgSchema.Properties[n])
			if err != nil {
				return "", err
			}
		}
	}
	return msgStructName, nil
}

// createGoStructForMessage creates the Go struct for a given Hedwig message
func createGoStructForMessage(
	sourceBuilder *strings.Builder, goTypes map[*jsonschema.Schema]string, msgType,
	msgStructName, version string, msgSchema *jsonschema.Schema) error {

	docstring := fmt.Sprintf("// %s represents the data for Hedwig message %s v%s\n", msgStructName, msgType, version)

	err := createGoStruct(sourceBuilder, goTypes, msgStructName, docstring, msgSchema)
	if err != nil {
		return err
	}
	_, err = sourceBuilder.WriteString(fmt.Sprintf(
		"// New%sData creates a new %s struct\n", msgStructName, msgStructName))
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}
	_, err = sourceBuilder.WriteString("// this method can be used as NewData when registering callback\n")
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}
	_, err = sourceBuilder.WriteString(
		fmt.Sprintf("func New%sData() interface{} { return new(%s) }\n\n", msgStructName, msgStructName))
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}
	return nil
}

// generate generates Go code for a given Hedwig schema
func generate(schemaFile, packageName, outputFile string, customFormats []string) error {
	rawSchema, err := ioutil.ReadFile(schemaFile)
	if err != nil {
		return errors.New("can't read schema")
	}

	var parsedSchema map[string]interface{}
	err = json.Unmarshal(rawSchema, &parsedSchema)
	if err != nil {
		return errors.New("can't read schema as JSON")
	}

	for _, customFormat := range customFormats {
		formats.Register(customFormat, func(in string) bool {
			return true
		})
	}

	sourceBuilder := &strings.Builder{}
	goTypes := make(map[*jsonschema.Schema]string)

	_, err = sourceBuilder.WriteString(autoGenHeader)
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}

	_, err = sourceBuilder.WriteString(fmt.Sprintf("package %s\n\n", packageName))
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}

	schemas := parsedSchema["schemas"].(map[string]interface{})
	msgTypes := keysGeneric(schemas)
	sort.Strings(msgTypes)

	msgStructNames := make(map[string]string)
	msgSchemas := make(map[string]*jsonschema.Schema)

	_, err = sourceBuilder.WriteString("/**** BEGIN base definitions ****/\n\n")
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}

	compiler := jsonschema.NewCompiler()
	// Force to draft version 4
	compiler.Draft = jsonschema.Draft4
	compiler.ExtractAnnotations = true

	err = compiler.AddResource(parsedSchema["id"].(string), strings.NewReader(string(rawSchema)))
	if err != nil {
		return errors.Wrap(err, "failed to add root resource")
	}

	for _, msgType := range msgTypes {
		versionSchemas := schemas[msgType].(map[string]interface{})
		versions := keysGeneric(versionSchemas)
		sort.Strings(versions)

		for _, version := range versions {
			msgSchemaStruct := versionSchemas[version]
			majorVersion, err := strconv.Atoi(strings.Replace(version, ".*", "", -1))
			if err != nil {
				return errors.Wrapf(err, "invalid version: %s", version)
			}

			schemaURL := fmt.Sprintf("%s/schemas/%s/%s", parsedSchema["id"], msgType, version)

			schemaByte, err := json.Marshal(msgSchemaStruct)
			if err != nil {
				return errors.Wrap(err, "failed marshaling")
			}

			err = compiler.AddResource(schemaURL, strings.NewReader(string(schemaByte)))
			if err != nil {
				return errors.Wrap(err, "failed to compile msg schema with error")
			}

			msgSchema, err := compiler.Compile(schemaURL)
			if err != nil {
				return errors.Wrap(err, "failed to read schema")
			}

			msgStructName, err := createGoTypesForMessage(sourceBuilder, goTypes, msgType, majorVersion, msgSchema)
			if err != nil {
				return err
			}
			msgStructNames[msgKey(msgType, version)] = msgStructName
			msgSchemas[msgKey(msgType, version)] = msgSchema
		}
	}

	_, err = sourceBuilder.WriteString("/**** END base definitions ****/\n\n")
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}
	_, err = sourceBuilder.WriteString("/**** BEGIN schema definitions ****/\n\n")
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}

	for _, msgType := range msgTypes {
		versionSchemas := schemas[msgType].(map[string]interface{})
		versions := keysGeneric(versionSchemas)
		sort.Strings(versions)

		for _, version := range versions {
			msgSchema := msgSchemas[msgKey(msgType, version)]
			msgStructName := msgStructNames[msgKey(msgType, version)]

			err := createGoStructForMessage(sourceBuilder, goTypes, msgType, msgStructName, version, msgSchema)
			if err != nil {
				return err
			}
		}
	}
	_, err = sourceBuilder.WriteString("/**** END schema definitions ****/")
	if err != nil {
		return errors.Wrap(err, "unable to write")
	}

	goSource := sourceBuilder.String()
	goSourceFormatted, err := format.Source([]byte(goSource))
	if err != nil {
		return errors.Wrap(err, "unable to format")
	}

	if outputFile == "" {
		fmt.Print(string(goSourceFormatted))
		return nil
	}

	err = ioutil.WriteFile(outputFile, goSourceFormatted, 0644)
	if err != nil {
		return errors.Wrap(err, "unable to write to output path")
	}
	return nil
}
