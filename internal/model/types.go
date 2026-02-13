package model

import "time"

type OutputFormat string

const (
	OutputTable OutputFormat = "table"
	OutputJSON  OutputFormat = "json"
	OutputWide  OutputFormat = "wide"
)

type Feed struct {
	ID            int64      `json:"id"`
	URL           string     `json:"url"`
	SiteURL       string     `json:"site_url,omitempty"`
	Title         string     `json:"title,omitempty"`
	Description   string     `json:"description,omitempty"`
	LastFetchedAt *time.Time `json:"last_fetched_at,omitempty"`
	ETag          string     `json:"etag,omitempty"`
	LastModified  string     `json:"last_modified,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	ErrorCount    int        `json:"error_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UnreadCount   int        `json:"unread_count"`
	TotalCount    int        `json:"total_count"`
}

type Entry struct {
	ID           int64      `json:"id"`
	FeedID       int64      `json:"feed_id"`
	FeedTitle    string     `json:"feed_title"`
	GUID         string     `json:"guid"`
	URL          string     `json:"url,omitempty"`
	ExternalURL  string     `json:"external_url,omitempty"`
	Title        string     `json:"title,omitempty"`
	Summary      string     `json:"summary,omitempty"`
	ContentHTML  string     `json:"content_html,omitempty"`
	ContentMD    string     `json:"content_md,omitempty"`
	Author       string     `json:"author,omitempty"`
	PublishedAt  *time.Time `json:"published_at,omitempty"`
	DateModified *time.Time `json:"date_modified,omitempty"`
	FetchedAt    time.Time  `json:"fetched_at"`
	Read         bool       `json:"read"`
	Starred      bool       `json:"starred"`
}

type Stats struct {
	Feeds   int `json:"feeds"`
	Unread  int `json:"unread"`
	Starred int `json:"starred"`
	Total   int `json:"total"`
}

type FetchResult struct {
	FeedID      int64  `json:"feed_id"`
	FeedTitle   string `json:"feed_title"`
	FeedURL     string `json:"feed_url"`
	NewEntries  int    `json:"new_entries"`
	Updated     int    `json:"updated_entries"`
	NotModified bool   `json:"not_modified"`
	Error       string `json:"error,omitempty"`
}

type FetchReport struct {
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Results   []FetchResult `json:"results"`
	Warnings  []string      `json:"warnings,omitempty"`
}

type EntryListOptions struct {
	Status string
	FeedID int64
	Limit  int
}

type SearchOptions struct {
	Query string
	Feed  int64
	Limit int
}

type UpsertEntryInput struct {
	FeedID       int64
	GUID         string
	URL          string
	ExternalURL  string
	Title        string
	Summary      string
	ContentHTML  string
	ContentMD    string
	Author       string
	PublishedAt  *time.Time
	DateModified *time.Time
}
