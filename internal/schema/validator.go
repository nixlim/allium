// Package schema provides JSON Schema validation for Allium specification files.
package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed all:schemas
var schemaFS embed.FS

// SchemaError represents a single schema validation error.
type SchemaError struct {
	Path       string `json:"path"`
	Message    string `json:"message"`
	ParseError bool   `json:"-"` // true when the error is a JSON parse or read failure
}

func (e SchemaError) String() string {
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

// SchemaValidator validates Allium JSON documents against the embedded JSON schemas.
type SchemaValidator struct {
	schema *jsonschema.Schema
}

// NewSchemaValidator creates a new validator with the embedded schemas loaded.
func NewSchemaValidator() (*SchemaValidator, error) {
	c := jsonschema.NewCompiler()

	// Walk all embedded schema files and add them to the compiler.
	// Use relative paths from schemas/v1/ as resource URLs so that
	// $ref resolution in the root schema (e.g. "definitions/common.json")
	// finds the correct resources.
	const schemaRoot = "schemas/v1/"
	err := fs.WalkDir(schemaFS, "schemas", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		data, err := schemaFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded schema %s: %w", path, err)
		}

		var schemaDoc any
		if err := json.Unmarshal(data, &schemaDoc); err != nil {
			return fmt.Errorf("parse embedded schema %s: %w", path, err)
		}

		// Use the relative path from the schema root directory so that
		// $ref paths like "definitions/common.json" resolve correctly.
		id := strings.TrimPrefix(path, schemaRoot)

		if err := c.AddResource(id, schemaDoc); err != nil {
			return fmt.Errorf("add schema resource %s (id=%s): %w", path, id, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load embedded schemas: %w", err)
	}

	schema, err := c.Compile("allium-spec.json")
	if err != nil {
		return nil, fmt.Errorf("compile root schema: %w", err)
	}

	return &SchemaValidator{schema: schema}, nil
}

// Validate validates an Allium JSON document at the given path against the schema.
func (v *SchemaValidator) Validate(docPath string) []SchemaError {
	data, err := os.ReadFile(docPath)
	if err != nil {
		return []SchemaError{{Message: fmt.Sprintf("failed to read file: %v", err), ParseError: true}}
	}

	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return []SchemaError{{Message: fmt.Sprintf("failed to parse JSON: %v", err), ParseError: true}}
	}

	return v.ValidateDocument(doc)
}

// ValidateDocument validates an already-parsed JSON document against the schema.
func (v *SchemaValidator) ValidateDocument(doc any) []SchemaError {
	err := v.schema.Validate(doc)
	if err == nil {
		return nil
	}

	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return []SchemaError{{Message: err.Error()}}
	}

	return collectErrors(validationErr)
}

// collectErrors recursively collects all leaf validation errors from a ValidationError.
func collectErrors(ve *jsonschema.ValidationError) []SchemaError {
	var errors []SchemaError

	instancePath := "/" + strings.Join(ve.InstanceLocation, "/")
	if len(ve.InstanceLocation) == 0 {
		instancePath = ""
	}

	if len(ve.Causes) == 0 {
		msg := ve.Error()
		if msg != "" {
			errors = append(errors, SchemaError{
				Path:    instancePath,
				Message: msg,
			})
		}
	} else {
		for _, cause := range ve.Causes {
			errors = append(errors, collectErrors(cause)...)
		}
	}

	return errors
}
