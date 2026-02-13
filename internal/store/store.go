package store

import (
	"database/sql"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFeedRow(scanner rowScanner) (Feed, error) {
	var f Feed
	var siteURL, title, desc, lastFetched, etag, lastMod, lastErr sql.NullString
	var createdAt string
	if err := scanner.Scan(
		&f.ID,
		&f.URL,
		&siteURL,
		&title,
		&desc,
		&lastFetched,
		&etag,
		&lastMod,
		&lastErr,
		&f.ErrorCount,
		&createdAt,
	); err != nil {
		return Feed{}, err
	}
	f.SiteURL = siteURL.String
	f.Title = title.String
	f.Description = desc.String
	f.ETag = etag.String
	f.LastModified = lastMod.String
	f.LastError = lastErr.String
	if t, err := parseDBTime(createdAt); err == nil {
		f.CreatedAt = t
	}
	if lastFetched.Valid {
		if t, err := parseDBTime(lastFetched.String); err == nil {
			f.LastFetchedAt = &t
		}
	}
	return f, nil
}

func scanFeedWithCountsRow(scanner rowScanner) (Feed, error) {
	var f Feed
	var siteURL, title, desc, lastFetched, etag, lastMod, lastErr sql.NullString
	var createdAt string
	if err := scanner.Scan(
		&f.ID,
		&f.URL,
		&siteURL,
		&title,
		&desc,
		&lastFetched,
		&etag,
		&lastMod,
		&lastErr,
		&f.ErrorCount,
		&createdAt,
		&f.UnreadCount,
		&f.TotalCount,
	); err != nil {
		return Feed{}, err
	}
	f.SiteURL = siteURL.String
	f.Title = title.String
	f.Description = desc.String
	f.ETag = etag.String
	f.LastModified = lastMod.String
	f.LastError = lastErr.String
	if t, err := parseDBTime(createdAt); err == nil {
		f.CreatedAt = t
	}
	if lastFetched.Valid {
		if t, err := parseDBTime(lastFetched.String); err == nil {
			f.LastFetchedAt = &t
		}
	}
	return f, nil
}

func scanEntry(scanner rowScanner) (Entry, error) {
	var e Entry
	var feedTitle sql.NullString
	var url, externalURL, title, summary, contentHTML, contentMD, author sql.NullString
	var publishedAt, dateModified sql.NullString
	var fetchedAt string
	if err := scanner.Scan(
		&e.ID,
		&e.FeedID,
		&feedTitle,
		&e.GUID,
		&url,
		&externalURL,
		&title,
		&summary,
		&contentHTML,
		&contentMD,
		&author,
		&publishedAt,
		&dateModified,
		&fetchedAt,
		&e.Read,
		&e.Starred,
	); err != nil {
		return Entry{}, err
	}
	e.FeedTitle = feedTitle.String
	e.URL = url.String
	e.ExternalURL = externalURL.String
	e.Title = title.String
	e.Summary = summary.String
	e.ContentHTML = contentHTML.String
	e.ContentMD = contentMD.String
	e.Author = author.String
	if publishedAt.Valid {
		if t, err := parseDBTime(publishedAt.String); err == nil {
			e.PublishedAt = &t
		}
	}
	if dateModified.Valid {
		if t, err := parseDBTime(dateModified.String); err == nil {
			e.DateModified = &t
		}
	}
	if t, err := parseDBTime(fetchedAt); err == nil {
		e.FetchedAt = t
	}
	return e, nil
}

func timestampBeforeDays(days int) time.Time {
	return time.Now().UTC().AddDate(0, 0, -days)
}
