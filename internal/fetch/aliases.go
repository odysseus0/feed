package fetch

import (
	"github.com/odysseus0/feed/internal/config"
	"github.com/odysseus0/feed/internal/model"
	"github.com/odysseus0/feed/internal/store"
)

type Config = config.Config
type Store = store.Store
type Feed = model.Feed
type FetchResult = model.FetchResult
type FetchReport = model.FetchReport
type EntryListOptions = model.EntryListOptions
type UpsertEntryInput = model.UpsertEntryInput
