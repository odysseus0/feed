package cli

import "github.com/tengjizhang/feed/internal/model"

type OutputFormat = model.OutputFormat
type Feed = model.Feed
type Entry = model.Entry
type Stats = model.Stats
type FetchResult = model.FetchResult
type FetchReport = model.FetchReport
type EntryListOptions = model.EntryListOptions
type SearchOptions = model.SearchOptions

const (
	OutputTable = model.OutputTable
	OutputJSON  = model.OutputJSON
	OutputWide  = model.OutputWide
)
