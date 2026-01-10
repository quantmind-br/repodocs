package domain

// CommonOptions contains shared configuration options for strategies and orchestration.
type CommonOptions struct {
	Verbose  bool
	DryRun   bool
	Force    bool
	RenderJS bool
	Limit    int
	Sync     bool
	FullSync bool
	Prune    bool
}

// DefaultCommonOptions returns CommonOptions with default values.
func DefaultCommonOptions() CommonOptions {
	return CommonOptions{}
}
