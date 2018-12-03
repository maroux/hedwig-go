package main

import (
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

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
	app.Version = version
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
