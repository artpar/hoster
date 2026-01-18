package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// SubstituteVariables Tests
// =============================================================================

func TestSubstituteVariables_Simple(t *testing.T) {
	vars := map[string]string{"DB_HOST": "localhost"}
	result := SubstituteVariables("${DB_HOST}", vars)
	assert.Equal(t, "localhost", result)
}

func TestSubstituteVariables_WithDefault_Found(t *testing.T) {
	vars := map[string]string{"PORT": "3000"}
	result := SubstituteVariables("${PORT:-8080}", vars)
	assert.Equal(t, "3000", result)
}

func TestSubstituteVariables_WithDefault_NotFound(t *testing.T) {
	vars := map[string]string{}
	result := SubstituteVariables("${PORT:-8080}", vars)
	assert.Equal(t, "8080", result)
}

func TestSubstituteVariables_NotFound_NoDefault(t *testing.T) {
	vars := map[string]string{}
	result := SubstituteVariables("${MISSING}", vars)
	assert.Equal(t, "${MISSING}", result) // Returns original
}

func TestSubstituteVariables_Multiple(t *testing.T) {
	vars := map[string]string{"HOST": "db", "PORT": "5432"}
	result := SubstituteVariables("postgres://${HOST}:${PORT}", vars)
	assert.Equal(t, "postgres://db:5432", result)
}

func TestSubstituteVariables_NoPlaceholders(t *testing.T) {
	vars := map[string]string{"KEY": "value"}
	result := SubstituteVariables("plain text", vars)
	assert.Equal(t, "plain text", result)
}

func TestSubstituteVariables_EmptyDefault(t *testing.T) {
	vars := map[string]string{}
	result := SubstituteVariables("${EMPTY:-}", vars)
	assert.Equal(t, "", result)
}

func TestSubstituteVariables_NilVariables(t *testing.T) {
	result := SubstituteVariables("${VAR:-default}", nil)
	assert.Equal(t, "default", result)
}

func TestSubstituteVariables_EmptyString(t *testing.T) {
	vars := map[string]string{"VAR": "value"}
	result := SubstituteVariables("", vars)
	assert.Equal(t, "", result)
}

func TestSubstituteVariables_MixedContent(t *testing.T) {
	vars := map[string]string{"APP_NAME": "myapp", "VERSION": "1.0"}
	result := SubstituteVariables("Starting ${APP_NAME} version ${VERSION}...", vars)
	assert.Equal(t, "Starting myapp version 1.0...", result)
}

func TestSubstituteVariables_PartialMatch(t *testing.T) {
	vars := map[string]string{"A": "1"}
	result := SubstituteVariables("${A}-${B:-2}", vars)
	assert.Equal(t, "1-2", result)
}

func TestSubstituteVariables_DefaultWithSpecialChars(t *testing.T) {
	vars := map[string]string{}
	result := SubstituteVariables("${URL:-http://localhost:8080/path}", vars)
	assert.Equal(t, "http://localhost:8080/path", result)
}

func TestSubstituteVariables_UnderscoreInName(t *testing.T) {
	vars := map[string]string{"DB_CONNECTION_STRING": "postgres://localhost"}
	result := SubstituteVariables("${DB_CONNECTION_STRING}", vars)
	assert.Equal(t, "postgres://localhost", result)
}

func TestSubstituteVariables_NumbersInName(t *testing.T) {
	vars := map[string]string{"APP_V2_PORT": "9000"}
	result := SubstituteVariables("${APP_V2_PORT}", vars)
	assert.Equal(t, "9000", result)
}

func TestSubstituteVariables_AdjacentPlaceholders(t *testing.T) {
	vars := map[string]string{"A": "1", "B": "2"}
	result := SubstituteVariables("${A}${B}", vars)
	assert.Equal(t, "12", result)
}

func TestSubstituteVariables_ValueWithDollarSign(t *testing.T) {
	vars := map[string]string{"PRICE": "$100"}
	result := SubstituteVariables("Cost: ${PRICE}", vars)
	assert.Equal(t, "Cost: $100", result)
}

func TestSubstituteVariables_EmptyValue(t *testing.T) {
	vars := map[string]string{"EMPTY": ""}
	result := SubstituteVariables("Value: [${EMPTY}]", vars)
	assert.Equal(t, "Value: []", result)
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestSubstituteVariables_TableDriven(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		variables map[string]string
		want      string
	}{
		{
			name:      "simple substitution",
			value:     "${VAR}",
			variables: map[string]string{"VAR": "value"},
			want:      "value",
		},
		{
			name:      "with default, var exists",
			value:     "${VAR:-default}",
			variables: map[string]string{"VAR": "actual"},
			want:      "actual",
		},
		{
			name:      "with default, var missing",
			value:     "${VAR:-default}",
			variables: map[string]string{},
			want:      "default",
		},
		{
			name:      "no placeholder",
			value:     "plain text",
			variables: map[string]string{},
			want:      "plain text",
		},
		{
			name:      "missing var no default",
			value:     "${MISSING}",
			variables: map[string]string{},
			want:      "${MISSING}",
		},
		{
			name:      "multiple vars",
			value:     "${A}-${B}-${C}",
			variables: map[string]string{"A": "1", "B": "2", "C": "3"},
			want:      "1-2-3",
		},
		{
			name:      "url pattern",
			value:     "${PROTOCOL:-http}://${HOST}:${PORT:-80}",
			variables: map[string]string{"HOST": "localhost"},
			want:      "http://localhost:80",
		},
		{
			name:      "database url",
			value:     "postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT:-5432}/${DB_NAME}",
			variables: map[string]string{"DB_USER": "admin", "DB_PASS": "secret", "DB_HOST": "db", "DB_NAME": "app"},
			want:      "postgres://admin:secret@db:5432/app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteVariables(tt.value, tt.variables)
			assert.Equal(t, tt.want, got)
		})
	}
}
