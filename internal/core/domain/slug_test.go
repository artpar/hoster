package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Slugify Tests
// =============================================================================

func TestSlugify_Basic(t *testing.T) {
	result := Slugify("Hello World")
	assert.Equal(t, "hello-world", result)
}

func TestSlugify_Lowercase(t *testing.T) {
	result := Slugify("already lowercase")
	assert.Equal(t, "already-lowercase", result)
}

func TestSlugify_Uppercase(t *testing.T) {
	result := Slugify("UPPERCASE NAME")
	assert.Equal(t, "uppercase-name", result)
}

func TestSlugify_MixedCase(t *testing.T) {
	result := Slugify("WordPress Blog")
	assert.Equal(t, "wordpress-blog", result)
}

func TestSlugify_WithNumbers(t *testing.T) {
	result := Slugify("Test123")
	assert.Equal(t, "test123", result)
}

func TestSlugify_RemovesSpecialChars(t *testing.T) {
	result := Slugify("My App!")
	assert.Equal(t, "my-app", result)
}

func TestSlugify_RemovesPunctuation(t *testing.T) {
	result := Slugify("hello, world.")
	assert.Equal(t, "hello-world", result)
}

func TestSlugify_PreservesHyphens(t *testing.T) {
	result := Slugify("my-app-name")
	assert.Equal(t, "my-app-name", result)
}

func TestSlugify_EmptyString(t *testing.T) {
	result := Slugify("")
	assert.Equal(t, "", result)
}

func TestSlugify_OnlySpecialChars(t *testing.T) {
	result := Slugify("!@#$%^&*()")
	assert.Equal(t, "", result)
}

func TestSlugify_MultipleSpaces(t *testing.T) {
	result := Slugify("hello   world")
	assert.Equal(t, "hello---world", result)
}

func TestSlugify_LeadingTrailingSpaces(t *testing.T) {
	result := Slugify(" trim me ")
	assert.Equal(t, "-trim-me-", result)
}

func TestSlugify_Numbers(t *testing.T) {
	result := Slugify("123")
	assert.Equal(t, "123", result)
}

func TestSlugify_NumbersAndLetters(t *testing.T) {
	result := Slugify("App2Go v3.0")
	assert.Equal(t, "app2go-v30", result)
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestSlugify_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic", "Hello World", "hello-world"},
		{"lowercase", "already lowercase", "already-lowercase"},
		{"uppercase", "UPPERCASE", "uppercase"},
		{"mixed", "MiXeD CaSe", "mixed-case"},
		{"numbers", "Test123App", "test123app"},
		{"special chars", "Hello! World?", "hello-world"},
		{"hyphens preserved", "my-app", "my-app"},
		{"empty", "", ""},
		{"unicode removed", "Hllo Wrld", "hllo-wrld"},
		{"underscores removed", "hello_world", "helloworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Slugify(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
