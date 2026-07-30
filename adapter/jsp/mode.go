package jsp

// ShellMode controls how the JSP operations build their payload.
type ShellMode int

const (
	// ShellStatic uses a pre-deployed shell with all operations built in.
	// The payload is a single action code. Works on ALL JDK versions.
	ShellStatic ShellMode = iota

	// ShellDynamic uses a ScriptEngine-based shell that evaluates inline
	// JavaScript code. The payload is JavaScript code. Requires Nashorn
	// (JDK 6-14, removed in JDK 15+).
	// Deprecated: ShellDynamic uses Nashorn (ScriptEngine) which was removed
	// in JDK 15 (2020). Only works on JDK 6-14. Use ShellStatic instead.
	ShellDynamic // Deprecated
)

// String returns a human-readable name for the mode.
func (m ShellMode) String() string {
	switch m {
	case ShellStatic:
		return "static"
	case ShellDynamic:
		return "dynamic"
	default:
		return "unknown"
	}
}
