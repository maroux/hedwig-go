package main

import (
	"encoding/json"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema"
	"github.com/santhosh-tekuri/jsonschema/formats"
	"github.com/urfave/cli"
)

// name creates the struct name for a given Hedwig message
func name(id string, majorVersion int, multipleMajorVersions bool) string {
	base := strcase.ToCamel(strings.Replace(id, ".", "_", -1))
	if !multipleMajorVersions {
		return base
	}
	return fmt.Sprintf("%sV%d", base, majorVersion)
}

// toCamel converts a go name for struct, variable, field, ... to appropriate camel case
func toCamel(s string) string {
	camel := strcase.ToCamel(s)
	if camel == "Id" {
		return "ID"
	}
	return camel
}

// getJsType gets the JSON Schema type given a list of types defined in the schema
func getJsType(types []string) string {
	jsType := ""
	if len(types) == 1 {
		return types[0]
	} else if len(types) == 2 {
		if types[0] == "null" {
			jsType = types[1]
		} else if types[1] == "null" {
			jsType = types[0]
		}
	}
	return jsType
}

// nullAllowed determines if a given JSON Schema type allows null values
func nullAllowed(sch *jsonschema.Schema) bool {
	types := sch.Types
	if sch.Ref != nil {
		types = sch.Ref.Types
	}
	for _, t := range types {
		if t == "null" {
			return true
		}
	}
	return false
}

// isRequired determines if a given property is required per JSON Schema
func isRequired(prop string, sch *jsonschema.Schema) bool {
	for _, p := range sch.Required {
		if p == prop {
			return true
		}
	}
	return false
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
	case "null":
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

// fieldDocstring returns Go docstring for a JSON Schema property
func fieldDocstring(sch *jsonschema.Schema, indent, name string) string {
	if sch.Description == "" {
		return ""
	}
	cleaned := strings.Replace(strings.TrimSpace(sch.Description), "\n", "\n\t// ", -1)
	return fmt.Sprintf("%s// %s - %s\n", indent, name, cleaned)
}

// createGoStruct creates a Go struct to represent a Hedwig message
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
	multipleMajorVersions bool, msgSchema *jsonschema.Schema) (string, error) {

	msgStructName := name(msgType, majorVersion, multipleMajorVersions)
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

// msgKey struct represents a Hedwig message to be used in maps
func msgKey(msgType, version string) string {
	return fmt.Sprintf("%s.%s", msgType, version)
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

		majorVersions := make(map[int]int)
		for version := range versionSchemas {

			majorVersion, err := strconv.Atoi(strings.Replace(version, ".*", "", -1))
			if err != nil {
				return errors.Errorf("failed to read schema: bad version: '%s' %s", version, err)
			}

			if _, ok := majorVersions[majorVersion]; !ok {
				majorVersions[majorVersion] = 1
			} else {
				majorVersions[majorVersion] += 1
			}
		}

		multipleMajorVersions := len(majorVersions) > 1

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

			msgStructName, err := createGoTypesForMessage(
				sourceBuilder, goTypes, msgType, majorVersion, multipleMajorVersions, msgSchema)
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

func main() {
	const (
		// customFormatsFlag represents the cli flag for custom formats. Defaults to empty list.
		customFormatsFlag = "custom-format"

		// packageFlag represents the cli flag for output module name
		packageFlag = "module"

		// outputFileFlag represents the cli flag for output module path. Defaults to stdout.
		outputFileFlag = "output-file"

		// schemaFileFlag represents the cli flag for Go package name for generated code
		schemaFileFlag = "schema-file"
	)

	app := cli.NewApp()
	app.Name = "Hedwig Go Models Generator"
	app.Usage = "Generate Go models for publishing/receiving Hedwig messages"
	app.Version = "0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  packageFlag,
			Usage: "Go package name to generate",
			Value: "hedwig",
		},
		cli.StringFlag{
			Name:  schemaFileFlag,
			Usage: "Schema file path",
		},
		cli.StringFlag{
			Name:  outputFileFlag,
			Usage: "Output file path",
		},
		cli.StringSliceFlag{
			Name: customFormatsFlag,
			Usage: "Custom JSON Schema formats to register. These won't be validated by JSON Schema, " +
				"but blindly allowed",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.String(schemaFileFlag) == "" {
			return errors.New("Schema file is required")
		}
		return generate(
			c.String(schemaFileFlag), c.String(packageFlag), c.String(outputFileFlag), c.StringSlice(customFormatsFlag))
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
