package resource

// BaseDownloadOptions contains common options shared by all resource download operations.
type BaseDownloadOptions struct {
	All        bool   // Download all resources
	Update     bool   // Update existing resources from local files
	OutputPath string // Custom output path pattern (overrides settings)
	Team       string // Filter by team tag (convenience flag for team:x)
	Tags       string // Comma-separated list of tags to filter by
	IDs        string // Comma-separated list of resource IDs to download
}
