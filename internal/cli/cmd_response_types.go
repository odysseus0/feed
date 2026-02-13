package cli

import "github.com/odysseus0/feed/internal/model"

type AddFeedResponse struct {
	Feed          model.Feed        `json:"feed"`
	Inserted      bool              `json:"inserted"`
	DiscoveredURL string            `json:"discovered_url"`
	FetchReport   model.FetchReport `json:"fetch_report"`
}

type RemoveFeedResponse struct {
	RemovedFeedID int64 `json:"removed_feed_id"`
}

type UpdateEntryResponse struct {
	EntryID int64 `json:"entry_id"`
	Read    *bool `json:"read,omitempty"`
	Starred *bool `json:"starred,omitempty"`
}

type BatchUpdateEntriesResponse struct {
	Updated int     `json:"updated"`
	IDs     []int64 `json:"ids"`
	Read    *bool   `json:"read,omitempty"`
	Starred *bool   `json:"starred,omitempty"`
}
