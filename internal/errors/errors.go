package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the category of build error
type ErrorType string

const (
	ParseError    ErrorType = "PARSE"
	TemplateError ErrorType = "TEMPLATE"
	LinkError     ErrorType = "LINK"
	AssetError    ErrorType = "ASSET"
	ConfigError   ErrorType = "CONFIG"
)

// BuildError represents a structured build error
type BuildError struct {
	Type       ErrorType
	File       string
	Line       int
	Column     int
	Message    string
	Suggestion string
	Context    string // Surrounding lines
}

// Error implements the error interface
func (e *BuildError) Error() string {
	loc := e.File
	if e.Line > 0 {
		loc = fmt.Sprintf("%s:%d", e.File, e.Line)
		if e.Column > 0 {
			loc = fmt.Sprintf("%s:%d:%d", e.File, e.Line, e.Column)
		}
	}
	return fmt.Sprintf("%s error in %s: %s", e.Type, loc, e.Message)
}

// FormatError formats a build error with context for human-readable output
func FormatError(err *BuildError) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("ERROR [%s]: %s\n", err.Type, err.Message))

	// Location
	if err.File != "" {
		loc := err.File
		if err.Line > 0 {
			loc = fmt.Sprintf("%s:%d", err.File, err.Line)
			if err.Column > 0 {
				loc = fmt.Sprintf("%s:%d:%d", err.File, err.Line, err.Column)
			}
		}
		sb.WriteString(fmt.Sprintf("  → %s\n", loc))
	}

	// Context
	if err.Context != "" {
		sb.WriteString("\n")
		lines := strings.Split(err.Context, "\n")
		for i, line := range lines {
			lineNum := err.Line - len(lines)/2 + i
			if lineNum == err.Line {
				sb.WriteString(fmt.Sprintf("  > %4d | %s\n", lineNum, line))
				if err.Column > 0 {
					pointer := strings.Repeat(" ", err.Column+8) + "^"
					sb.WriteString(fmt.Sprintf("         %s\n", pointer))
				}
			} else {
				sb.WriteString(fmt.Sprintf("    %4d | %s\n", lineNum, line))
			}
		}
	}

	// Suggestion
	if err.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\n  💡 %s\n", err.Suggestion))
	}

	return sb.String()
}

// FormatErrors formats multiple build errors
func FormatErrors(errs []*BuildError) string {
	if len(errs) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n%d build error(s):\n", len(errs)))
	sb.WriteString(strings.Repeat("─", 50) + "\n")

	for _, err := range errs {
		sb.WriteString(FormatError(err))
		sb.WriteString(strings.Repeat("─", 50) + "\n")
	}

	return sb.String()
}

// NewParseError creates a parse error
func NewParseError(file string, line int, msg, suggestion string) *BuildError {
	return &BuildError{
		Type:       ParseError,
		File:       file,
		Line:       line,
		Message:    msg,
		Suggestion: suggestion,
	}
}

// NewConfigError creates a config error
func NewConfigError(msg, suggestion string) *BuildError {
	return &BuildError{
		Type:       ConfigError,
		File:       "config.yaml",
		Message:    msg,
		Suggestion: suggestion,
	}
}

// NewAssetError creates an asset processing error
func NewAssetError(file, msg string) *BuildError {
	return &BuildError{
		Type:    AssetError,
		File:    file,
		Message: msg,
	}
}

// NewLinkError creates a link validation error
func NewLinkError(file string, line int, url, msg string) *BuildError {
	return &BuildError{
		Type:    LinkError,
		File:    file,
		Line:    line,
		Message: fmt.Sprintf("broken link %s: %s", url, msg),
	}
}
