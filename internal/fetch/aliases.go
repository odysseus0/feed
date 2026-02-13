package fetch

import (
	"github.com/tengjizhang/feed/internal/config"
	"github.com/tengjizhang/feed/internal/model"
	"github.com/tengjizhang/feed/internal/store"
)

type Config = config.Config
type Store = store.Store
type Feed = model.Feed
type FetchResult = model.FetchResult
type FetchReport = model.FetchReport
type EntryListOptions = model.EntryListOptions
type UpsertEntryInput = model.UpsertEntryInput
