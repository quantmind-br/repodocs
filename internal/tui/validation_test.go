package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid_string", input: "hello", wantErr: false},
		{name: "empty_string", input: "", wantErr: true},
		{name: "whitespace_only", input: "   ", wantErr: true},
		{name: "string_with_spaces", input: "  hello  ", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid_seconds", input: "30s", wantErr: false},
		{name: "valid_minutes", input: "5m", wantErr: false},
		{name: "valid_hours", input: "1h", wantErr: false},
		{name: "valid_complex", input: "1h30m45s", wantErr: false},
		{name: "empty_string", input: "", wantErr: false},
		{name: "invalid_format", input: "30", wantErr: true},
		{name: "invalid_unit", input: "30x", wantErr: true},
		{name: "whitespace_only", input: "   ", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDuration(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid_positive", input: "5", wantErr: false},
		{name: "valid_one", input: "1", wantErr: false},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "empty_string", input: "", wantErr: false},
		{name: "not_a_number", input: "abc", wantErr: true},
		{name: "float", input: "1.5", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePositiveInt(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateIntRange(t *testing.T) {
	validator := ValidateIntRange(1, 10)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "in_range_min", input: "1", wantErr: false},
		{name: "in_range_max", input: "10", wantErr: false},
		{name: "in_range_middle", input: "5", wantErr: false},
		{name: "below_min", input: "0", wantErr: true},
		{name: "above_max", input: "11", wantErr: true},
		{name: "empty_string", input: "", wantErr: false},
		{name: "not_a_number", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateFloat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid_integer", input: "5", wantErr: false},
		{name: "valid_float", input: "1.5", wantErr: false},
		{name: "valid_negative", input: "-1.5", wantErr: false},
		{name: "empty_string", input: "", wantErr: false},
		{name: "not_a_number", input: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFloat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateFloatRange(t *testing.T) {
	validator := ValidateFloatRange(0, 2)

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "in_range_min", input: "0", wantErr: false},
		{name: "in_range_max", input: "2", wantErr: false},
		{name: "in_range_middle", input: "1.0", wantErr: false},
		{name: "below_min", input: "-0.1", wantErr: true},
		{name: "above_max", input: "2.1", wantErr: true},
		{name: "empty_string", input: "", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLogLevel(t *testing.T) {
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	for _, level := range validLevels {
		t.Run("valid_"+level, func(t *testing.T) {
			assert.NoError(t, ValidateLogLevel(level))
		})
	}

	t.Run("invalid_level", func(t *testing.T) {
		assert.Error(t, ValidateLogLevel("invalid"))
	})
}

func TestValidateLogFormat(t *testing.T) {
	validFormats := []string{"json", "pretty", "text"}
	for _, format := range validFormats {
		t.Run("valid_"+format, func(t *testing.T) {
			assert.NoError(t, ValidateLogFormat(format))
		})
	}

	t.Run("invalid_format", func(t *testing.T) {
		assert.Error(t, ValidateLogFormat("invalid"))
	})
}

func TestValidateLLMProvider(t *testing.T) {
	validProviders := []string{"openai", "anthropic", "google", ""}
	for _, provider := range validProviders {
		name := provider
		if name == "" {
			name = "empty"
		}
		t.Run("valid_"+name, func(t *testing.T) {
			assert.NoError(t, ValidateLLMProvider(provider))
		})
	}

	t.Run("invalid_provider", func(t *testing.T) {
		assert.Error(t, ValidateLLMProvider("invalid"))
	})
}
