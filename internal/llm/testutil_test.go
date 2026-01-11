package llm

import (
	"encoding/json"
	"io"
)

// decodeJSON is a test helper function
func decodeJSON(r io.Reader, v interface{}) error {
	decoder := json.NewDecoder(r)
	return decoder.Decode(v)
}
