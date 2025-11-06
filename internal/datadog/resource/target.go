package resource

// Target represents a Datadog resource (dashboard, monitor, etc.) with its ID and file path.
// The generic type T allows this to work with both string IDs (dashboards) and int IDs (monitors).
type Target[T comparable] struct {
	ID   T              // Resource ID (string for dashboards, int for monitors)
	Path string         // File path where the resource should be written
	Data map[string]any // Full resource data from API (cached to avoid duplicate requests)
}

// TargetResult wraps a Target with a potential error from target generation.
type TargetResult[T comparable] struct {
	Target Target[T] // The resource target containing ID, path, and optional cached data
	Err    error     // Error encountered during target generation, if any
}
