package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCategoryByID(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{id: "output", expected: "Output"},
		{id: "concurrency", expected: "Concurrency"},
		{id: "cache", expected: "Cache"},
		{id: "rendering", expected: "Rendering"},
		{id: "stealth", expected: "Stealth"},
		{id: "logging", expected: "Logging"},
		{id: "llm", expected: "LLM"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			cat := GetCategoryByID(tt.id)
			assert.NotNil(t, cat)
			assert.Equal(t, tt.expected, cat.Name)
		})
	}

	t.Run("invalid_id", func(t *testing.T) {
		cat := GetCategoryByID("nonexistent")
		assert.Nil(t, cat)
	})
}

func TestGetCategoryNames(t *testing.T) {
	names := GetCategoryNames()

	assert.Len(t, names, len(Categories))
	assert.Contains(t, names, "Output")
	assert.Contains(t, names, "Concurrency")
	assert.Contains(t, names, "Cache")
	assert.Contains(t, names, "Rendering")
	assert.Contains(t, names, "Stealth")
	assert.Contains(t, names, "Logging")
	assert.Contains(t, names, "LLM")
}

func TestCategories(t *testing.T) {
	assert.Len(t, Categories, 7)

	for _, cat := range Categories {
		assert.NotEmpty(t, cat.ID)
		assert.NotEmpty(t, cat.Name)
		assert.NotEmpty(t, cat.Description)
	}
}
