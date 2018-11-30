/*
 * Copyright 2018, Automatic Inc.
 * All rights reserved.
 *
 * Author: Aniruddha Maru
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	tests := []struct {
		dir   string
		error string
	}{
		{
			dir: "simple",
		},
		{
			dir: "definitions",
		},
		{
			dir: "sub-objects",
		},
		{
			dir: "nullable",
		},
		{
			dir: "optional",
		},
		{
			dir: "multiple-majors",
		},
		{
			dir:   "invalid-jsonschema",
			error: "can't read schema",
		},
		{
			dir:   "invalid-schemas-prop",
			error: "failed to read schema: bad version: 'description' strconv.Atoi: parsing \"description\": invalid syntax",
		},
		{
			dir:   "invalid-refs",
			error: "can't handle schema reference",
		},
	}
	for _, test := range tests {
		t.Run(test.dir, func(subT *testing.T) {
			f, err := ioutil.TempFile("", "models.go.")
			require.NoError(subT, err)
			defer f.Close()
			defer os.Remove(f.Name())

			schemaFile := fmt.Sprintf("test-schemas/%s/schema.json", test.dir)
			err = generate(schemaFile, "hedwig", f.Name(), []string{})
			if test.error != "" {
				assert.Error(subT, err, test.error)
			} else {
				assert.NoError(subT, err)

				modelsFile := fmt.Sprintf("test-schemas/%s/models.go", test.dir)
				expected, err := ioutil.ReadFile(modelsFile)
				require.NoError(subT, err)

				generated, err := ioutil.ReadFile(f.Name())
				require.NoError(subT, err)

				assert.Equal(subT, string(expected), string(generated))
			}
		})
	}
}
