package judges

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// Validator valida resultados de judges contra JSON Schema.
type Validator struct{}

// NewValidator cria um novo validador.
func NewValidator() *Validator {
	return &Validator{}
}

// Validate valida um resultado JSON contra um schema JSON.
// Retorna nil se valido, ou erro com detalhes da validacao.
func (v *Validator) Validate(resultJSON, schemaJSON string) error {
	schemaLoader := gojsonschema.NewBytesLoader([]byte(schemaJSON))
	resultLoader := gojsonschema.NewBytesLoader([]byte(resultJSON))

	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("invalid schema: %w", err)
	}

	result, err := schema.Validate(resultLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, err := range result.Errors() {
			errors = append(errors, err.String())
		}
		return fmt.Errorf("schema validation failed: %v", errors)
	}

	return nil
}

// ValidateMap valida um map contra um schema JSON.
func (v *Validator) ValidateMap(result map[string]interface{}, schemaJSON string) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	return v.Validate(string(data), schemaJSON)
}
