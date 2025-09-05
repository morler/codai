package models

import "time"

// FileData holds the path and content of a file
type FileData struct {
	RelativePath   string
	Code           string
	TreeSitterCode string
}

type FullContextData struct {
	FileData []FileData
	RawCodes []string
}

// ProjectSnapshot represents a snapshot of project file states for incremental scanning
type ProjectSnapshot struct {
	RootDir   string                    `json:"root_dir"`
	Timestamp time.Time                 `json:"timestamp"`
	Files     map[string]FileSnapshot   `json:"files"`
}

// FileSnapshot represents the state of a single file
type FileSnapshot struct {
	RelativePath string    `json:"relative_path"`
	ModTime      time.Time `json:"mod_time"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
}
