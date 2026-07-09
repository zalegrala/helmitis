package interchange

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schema.json
var schemaJSON []byte

type schemaResult struct {
	schema *jsonschema.Schema
	err    error
}

var getSchema = sync.OnceValue(func() schemaResult {
	var raw interface{}
	if err := json.Unmarshal(schemaJSON, &raw); err != nil {
		return schemaResult{err: err}
	}
	c := jsonschema.NewCompiler()
	if err := c.AddResource("interchange.json", raw); err != nil {
		return schemaResult{err: err}
	}
	s, err := c.Compile("interchange.json")
	return schemaResult{schema: s, err: err}
})

// Validate checks raw interchange bytes against the embedded schema.
func Validate(data []byte) error {
	res := getSchema()
	if res.err != nil {
		return fmt.Errorf("compile schema: %w", res.err)
	}
	schema := res.schema
	var inst interface{}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&inst); err != nil {
		return fmt.Errorf("parse interchange JSON: %w", err)
	}
	if err := schema.Validate(inst); err != nil {
		return fmt.Errorf("interchange validation failed: %w", err)
	}
	return nil
}

// DecodeAndValidate validates then decodes into a Document.
func DecodeAndValidate(data []byte) (Document, error) {
	if err := Validate(data); err != nil {
		return Document{}, err
	}
	return Decode(data)
}
