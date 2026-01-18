package domain

// =============================================================================
// Slug Generation
// =============================================================================

// Slugify converts a name to a URL-safe slug.
//
// The transformation rules are:
//   - Lowercase letters (a-z) are kept as-is
//   - Digits (0-9) are kept as-is
//   - Hyphens (-) are kept as-is
//   - Uppercase letters (A-Z) are converted to lowercase
//   - Spaces are converted to hyphens
//   - All other characters are removed
//
// This is a pure function with no side effects.
//
// Example:
//
//	Slugify("Hello World")     // returns "hello-world"
//	Slugify("My App 2.0!")     // returns "my-app-20"
//	Slugify("WordPress Blog")  // returns "wordpress-blog"
func Slugify(name string) string {
	slug := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			slug += string(r)
		} else if r >= 'A' && r <= 'Z' {
			slug += string(r + 32) // convert to lowercase
		} else if r == ' ' {
			slug += "-"
		}
		// All other characters are dropped
	}
	return slug
}
