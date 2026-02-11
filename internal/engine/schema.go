// Package engine provides a schema-driven CRUD engine.
// Resources are defined as data (schema), and the engine interprets them
// to provide auto-CRUD store, REST API, state machine enforcement, and migrations.
package engine

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// FieldType represents the SQL/Go type of a field.
type FieldType int

const (
	TypeString    FieldType = iota // TEXT
	TypeText                       // TEXT (large)
	TypeInt                        // INTEGER
	TypeFloat                      // REAL
	TypeBool                       // INTEGER (0/1)
	TypeJSON                       // TEXT (JSON-encoded)
	TypeTimestamp                   // DATETIME
	TypeRef                        // INTEGER (FK to another entity)
	TypeSoftRef                    // TEXT (reference_id of another entity, not a FK)
)

// Field defines a single column in a resource.
type Field struct {
	Name         string
	Type         FieldType
	Required     bool
	Unique       bool
	Nullable     bool
	DefaultValue interface{} // nil means no default
	MinInt       *int64
	MaxInt       *int64
	MinLen       *int
	MaxLen       *int
	Pattern      *regexp.Regexp
	RefTable     string // For TypeRef/TypeSoftRef: target table name
	Computed     func(row map[string]interface{}) interface{}
	WriteOnly    bool // If true, never included in GET responses (e.g., private_key)
	Encrypted    bool // If true, value is encrypted at rest
	Internal     bool // If true, not settable via API (e.g., creator_id set from auth)
}

// GuardFunc checks whether a state transition is allowed given the current row.
type GuardFunc func(row map[string]interface{}) error

// StateMachine defines a state machine on a string field.
type StateMachine struct {
	Field       string                       // The column that holds the state
	Initial     string                       // Default state on create
	Transitions map[string][]string          // from → []to
	Guards      map[string]GuardFunc         // to-state → guard
	OnEnter     map[string]string            // to-state → command name
}

// CanTransition checks if transitioning from → to is allowed.
func (sm *StateMachine) CanTransition(from, to string) bool {
	allowed, ok := sm.Transitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// AllStates returns all unique states in the state machine.
func (sm *StateMachine) AllStates() []string {
	seen := map[string]bool{}
	var states []string
	for from, tos := range sm.Transitions {
		if !seen[from] {
			seen[from] = true
			states = append(states, from)
		}
		for _, to := range tos {
			if !seen[to] {
				seen[to] = true
				states = append(states, to)
			}
		}
	}
	return states
}

// CustomAction defines an action endpoint beyond standard CRUD.
type CustomAction struct {
	Name    string // e.g., "publish", "start", "stop"
	Method  string // HTTP method, e.g., "POST", "DELETE"
	// Handler is set at registration time
}

// VisibilityFunc determines whether a row is visible to the given auth context.
// Return true if the user can see this row.
type VisibilityFunc func(ctx context.Context, authCtx AuthContext, row map[string]interface{}) bool

// BeforeCreateFunc is called before creating a row. It can modify the data.
type BeforeCreateFunc func(ctx context.Context, authCtx AuthContext, data map[string]interface{}) error

// BeforeDeleteFunc is called before deleting a row. It can return an error to prevent deletion.
type BeforeDeleteFunc func(ctx context.Context, authCtx AuthContext, row map[string]interface{}) error

// Resource defines a complete entity.
type Resource struct {
	Name         string // table name, e.g., "templates"
	RefPrefix    string // prefix for reference_id, e.g., "tmpl_"
	Owner        string // field name that references the authenticated user's ID (e.g., "creator_id")
	Fields       []Field
	StateMachine *StateMachine
	Actions      []CustomAction

	// Authorization hooks
	Visibility   VisibilityFunc
	BeforeCreate BeforeCreateFunc
	BeforeDelete BeforeDeleteFunc

	// If true, list without auth returns all rows (e.g., published templates)
	PublicRead bool
}

// AuthContext is a minimal auth interface the engine needs.
type AuthContext struct {
	Authenticated bool
	UserID        int
	ReferenceID   string
	PlanID        string
	PlanLimits    PlanLimits
}

// FieldByName returns a field by name, or nil if not found.
func (r *Resource) FieldByName(name string) *Field {
	for i := range r.Fields {
		if r.Fields[i].Name == name {
			return &r.Fields[i]
		}
	}
	return nil
}

// =============================================================================
// Field builder helpers
// =============================================================================

func StringField(name string) Field {
	return Field{Name: name, Type: TypeString}
}

func TextField(name string) Field {
	return Field{Name: name, Type: TypeText}
}

func IntField(name string) Field {
	return Field{Name: name, Type: TypeInt}
}

func FloatField(name string) Field {
	return Field{Name: name, Type: TypeFloat}
}

func BoolField(name string) Field {
	return Field{Name: name, Type: TypeBool}
}

func JSONField(name string) Field {
	return Field{Name: name, Type: TypeJSON, Nullable: true}
}

func TimestampField(name string) Field {
	return Field{Name: name, Type: TypeTimestamp, Nullable: true}
}

func RefField(name, table string) Field {
	return Field{Name: name, Type: TypeRef, RefTable: table}
}

func SoftRefField(name, table string) Field {
	return Field{Name: name, Type: TypeSoftRef, RefTable: table, Nullable: true}
}

// WithRequired returns a copy of the field with Required=true.
func (f Field) WithRequired() Field { f.Required = true; return f }

// WithUnique returns a copy of the field with Unique=true.
func (f Field) WithUnique() Field { f.Unique = true; return f }

// WithNullable returns a copy of the field with Nullable=true.
func (f Field) WithNullable() Field { f.Nullable = true; return f }

// WithDefault returns a copy of the field with DefaultValue set.
func (f Field) WithDefault(v interface{}) Field { f.DefaultValue = v; return f }

// WithMin returns a copy of the field with minimum constraint.
func (f Field) WithMin(n int64) Field { f.MinInt = &n; return f }

// WithMax returns a copy of the field with maximum constraint.
func (f Field) WithMax(n int64) Field { f.MaxInt = &n; return f }

// WithMinLen returns a copy of the field with minimum length.
func (f Field) WithMinLen(n int) Field { f.MinLen = &n; return f }

// WithMaxLen returns a copy of the field with maximum length.
func (f Field) WithMaxLen(n int) Field { f.MaxLen = &n; return f }

// WithPattern returns a copy of the field with a regex pattern.
func (f Field) WithPattern(pattern string) Field {
	f.Pattern = regexp.MustCompile(pattern)
	return f
}

// WithComputed returns a copy of the field with a computed function.
func (f Field) WithComputed(fn func(row map[string]interface{}) interface{}) Field {
	f.Computed = fn
	return f
}

// WithWriteOnly marks the field as write-only (never in GET responses).
func (f Field) WithWriteOnly() Field { f.WriteOnly = true; return f }

// WithEncrypted marks the field as encrypted at rest.
func (f Field) WithEncrypted() Field { f.Encrypted = true; return f }

// WithInternal marks the field as internal (set by system, not API).
func (f Field) WithInternal() Field { f.Internal = true; return f }

// =============================================================================
// Guard helpers
// =============================================================================

// RequireField returns a guard that ensures a field is non-empty.
func RequireField(fieldName string) GuardFunc {
	return func(row map[string]interface{}) error {
		v, ok := row[fieldName]
		if !ok || v == nil || v == "" || v == 0 {
			return fmt.Errorf("%s is required for this transition", fieldName)
		}
		return nil
	}
}

// =============================================================================
// SQL type helpers
// =============================================================================

// SQLType returns the SQLite column type for this field type.
func (ft FieldType) SQLType() string {
	switch ft {
	case TypeString, TypeText, TypeSoftRef:
		return "TEXT"
	case TypeInt, TypeRef, TypeBool:
		return "INTEGER"
	case TypeFloat:
		return "REAL"
	case TypeJSON:
		return "TEXT" // JSON stored as text
	case TypeTimestamp:
		return "DATETIME"
	default:
		return "TEXT"
	}
}

// =============================================================================
// Migration generation
// =============================================================================

// GenerateCreateSQL generates a CREATE TABLE statement for this resource.
func (r *Resource) GenerateCreateSQL() string {
	var cols []string

	// Standard columns
	cols = append(cols, "id INTEGER PRIMARY KEY AUTOINCREMENT")
	cols = append(cols, "reference_id TEXT UNIQUE NOT NULL")

	for _, f := range r.Fields {
		col := fmt.Sprintf("%s %s", f.Name, f.Type.SQLType())
		if !f.Nullable && f.DefaultValue == nil && f.Type != TypeJSON {
			col += " NOT NULL"
		}
		if f.Unique {
			col += " UNIQUE"
		}
		if f.DefaultValue != nil {
			col += fmt.Sprintf(" DEFAULT %s", sqlDefault(f.DefaultValue))
		}
		cols = append(cols, col)
	}

	// Standard timestamps
	cols = append(cols, "created_at DATETIME NOT NULL DEFAULT (datetime('now'))")
	cols = append(cols, "updated_at DATETIME NOT NULL DEFAULT (datetime('now'))")

	// FK constraints
	for _, f := range r.Fields {
		if f.Type == TypeRef && f.RefTable != "" {
			cols = append(cols, fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(id)", f.Name, f.RefTable))
		}
	}

	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n  %s\n)", r.Name, strings.Join(cols, ",\n  "))

	// Indexes
	var indexes []string
	for _, f := range r.Fields {
		if f.Type == TypeRef || f.Type == TypeSoftRef {
			indexes = append(indexes, fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s(%s)", r.Name, f.Name, r.Name, f.Name))
		}
	}

	if len(indexes) > 0 {
		sql += ";\n" + strings.Join(indexes, ";\n")
	}

	return sql
}

func sqlDefault(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("'%s'", val)
	case bool:
		if val {
			return "1"
		}
		return "0"
	case int, int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%f", val)
	default:
		return fmt.Sprintf("'%v'", val)
	}
}
