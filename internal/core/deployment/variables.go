package deployment

import "regexp"

// =============================================================================
// Variable Substitution Functions
// =============================================================================

// varPlaceholderRegex matches ${VAR} and ${VAR:-default} patterns.
// Groups:
//   - Group 1: Variable name (required)
//   - Group 2: Default value (optional, after :-)
var varPlaceholderRegex = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(?::-([^}]*))?\}`)

// SubstituteVariables replaces ${VAR} and ${VAR:-default} placeholders with values
// from the variables map.
//
// Behavior:
//   - ${VAR} - replaced with variables["VAR"] if exists, otherwise kept as-is
//   - ${VAR:-default} - replaced with variables["VAR"] if exists, otherwise "default"
//   - Unmatched text is left unchanged
//
// Examples:
//
//	SubstituteVariables("${DB_HOST}", map[string]string{"DB_HOST": "localhost"})
//	// Returns: "localhost"
//
//	SubstituteVariables("${PORT:-8080}", map[string]string{})
//	// Returns: "8080"
//
//	SubstituteVariables("${MISSING}", map[string]string{})
//	// Returns: "${MISSING}"
//
//	SubstituteVariables("postgres://${HOST}:${PORT}", map[string]string{"HOST": "db", "PORT": "5432"})
//	// Returns: "postgres://db:5432"
func SubstituteVariables(value string, variables map[string]string) string {
	if variables == nil {
		variables = make(map[string]string)
	}

	return varPlaceholderRegex.ReplaceAllStringFunc(value, func(match string) string {
		submatch := varPlaceholderRegex.FindStringSubmatch(match)
		if len(submatch) >= 2 {
			varName := submatch[1]
			if val, ok := variables[varName]; ok {
				return val
			}
			// Return default if specified (even empty string)
			if len(submatch) >= 3 && submatch[2] != "" {
				return submatch[2]
			}
			// Check for empty default case ${VAR:-}
			if len(submatch) >= 3 && len(match) > len(varName)+4 { // ${VAR:-} is longer than ${VAR}
				// If the regex matched and the original match contains ":-", return empty string
				if regexp.MustCompile(`\$\{` + varName + `:-\}`).MatchString(match) {
					return ""
				}
			}
		}
		return match // Return original if no substitution
	})
}
