package gori

// ProjectStatus tracks the status of a Git repository
type ProjectStatus struct {
	Path              string
	IsDirty           bool
	HasStash          bool
	Upstreamed        bool
	isDirtySnoozed    bool
	hasStashSnoozed   bool
	upstreamedSnoozed bool
	StatusString      string
}

func NewProject(path string, isDirty bool, hasStash bool, upstreamed bool) ProjectStatus {
	return ProjectStatus{
		Path:       path,
		IsDirty:    isDirty,
		HasStash:   hasStash,
		Upstreamed: upstreamed,
	}
}

func (p ProjectStatus) Clean() bool {
	return !(p.IsDirty || p.HasStash || !p.Upstreamed)
}
