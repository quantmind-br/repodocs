package state

import "time"

// StateVersion is the schema version for state file migration
const StateVersion = 1

// SyncState represents the complete synchronization state for a source
type SyncState struct {
	Version   int                  `json:"version"`
	SourceURL string               `json:"source_url"`
	Strategy  string               `json:"strategy,omitempty"`
	LastSync  time.Time            `json:"last_sync"`
	Pages     map[string]PageState `json:"pages"`
}

// PageState represents the state of an individual processed page
type PageState struct {
	ContentHash string    `json:"content_hash"`
	FetchedAt   time.Time `json:"fetched_at"`
	FilePath    string    `json:"file_path"`
}

// NewSyncState creates a new empty sync state
func NewSyncState(sourceURL, strategy string) *SyncState {
	return &SyncState{
		Version:   StateVersion,
		SourceURL: sourceURL,
		Strategy:  strategy,
		LastSync:  time.Now(),
		Pages:     make(map[string]PageState),
	}
}

// PageCount returns the number of pages in the state
func (s *SyncState) PageCount() int {
	return len(s.Pages)
}

// HasPage checks if a page exists in the state
func (s *SyncState) HasPage(url string) bool {
	_, exists := s.Pages[url]
	return exists
}

// GetPage returns a page state by URL
func (s *SyncState) GetPage(url string) (PageState, bool) {
	page, exists := s.Pages[url]
	return page, exists
}

// SetPage updates or adds a page to the state
func (s *SyncState) SetPage(url string, page PageState) {
	s.Pages[url] = page
}

// RemovePage removes a page from the state
func (s *SyncState) RemovePage(url string) {
	delete(s.Pages, url)
}
