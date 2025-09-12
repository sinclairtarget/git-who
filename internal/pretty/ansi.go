// ANSI escape codes
package pretty

var colorEnabled = true

// SetColorEnabled controls whether ANSI color codes are output
func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

// GetColorEnabled returns whether ANSI color codes are currently enabled
func GetColorEnabled() bool {
	return colorEnabled
}

const resetCode string = "\x1b[0m"
const greenCode string = "\x1b[32m"
const redCode string = "\x1b[31m"
const defaultColorCode string = "\x1b[39m"
const dimCode string = "\x1b[2m"
const invertCode string = "\x1b[7m"

// Reset returns the reset ANSI code if colors are enabled, empty string otherwise
func Reset() string {
	if colorEnabled {
		return resetCode
	}
	return ""
}

// Green returns the green ANSI code if colors are enabled, empty string otherwise
func Green() string {
	if colorEnabled {
		return greenCode
	}
	return ""
}

// Red returns the red ANSI code if colors are enabled, empty string otherwise
func Red() string {
	if colorEnabled {
		return redCode
	}
	return ""
}

// DefaultColor returns the default color ANSI code if colors are enabled, empty string otherwise
func DefaultColor() string {
	if colorEnabled {
		return defaultColorCode
	}
	return ""
}

// Dim returns the dim ANSI code if colors are enabled, empty string otherwise
func Dim() string {
	if colorEnabled {
		return dimCode
	}
	return ""
}

// Invert returns the invert ANSI code if colors are enabled, empty string otherwise
func Invert() string {
	if colorEnabled {
		return invertCode
	}
	return ""
}

// EraseLine always returns the erase line code as it's used for progress indicators
const EraseLine string = "\x1b[2K"
