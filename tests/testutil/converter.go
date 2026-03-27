package testutil

import (
	"testing"

	"github.com/quantmind-br/repodocs/internal/converter"
)

// NewHTMLConverter returns a converter pipeline suitable for unit tests.
func NewHTMLConverter(t *testing.T) *converter.Pipeline {
	t.Helper()
	return converter.NewPipeline(converter.PipelineOptions{})
}
