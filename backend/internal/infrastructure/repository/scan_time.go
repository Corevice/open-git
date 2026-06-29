package repository

import (
	"fmt"
	"time"
)

// sqliteTimestampFormats mirrors the layouts mattn/go-sqlite3 uses when it binds
// time.Time values. SQLite has no native timestamp type: the driver only
// auto-converts a column back to time.Time when the column's declared type is
// recognized (timestamp/datetime/date). Columns declared as TIMESTAMPTZ (used
// for PostgreSQL parity) are therefore returned as raw strings under SQLite and
// must be parsed manually.
var sqliteTimestampFormats = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",
}

// nullTime is a driver-portable scan target for nullable timestamp columns.
//
//   - PostgreSQL (lib/pq): TIMESTAMP/TIMESTAMPTZ values arrive as time.Time.
//   - SQLite (mattn/go-sqlite3): TIMESTAMPTZ values arrive as a string/[]byte
//     because the declared type is not recognized for auto-conversion.
//
// Both forms are handled here so the same repository code works against both
// databases. It exposes the same Time/Valid fields as sql.NullTime so existing
// call sites need only swap the type.
type nullTime struct {
	Time  time.Time
	Valid bool
}

func (n *nullTime) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		n.Time, n.Valid = time.Time{}, false
		return nil
	case time.Time:
		n.Time, n.Valid = v, true
		return nil
	case []byte:
		return n.scanString(string(v))
	case string:
		return n.scanString(v)
	default:
		return fmt.Errorf("nullTime: unsupported scan type %T", src)
	}
}

func (n *nullTime) scanString(s string) error {
	if s == "" {
		n.Time, n.Valid = time.Time{}, false
		return nil
	}
	for _, layout := range sqliteTimestampFormats {
		if t, err := time.Parse(layout, s); err == nil {
			n.Time, n.Valid = t, true
			return nil
		}
	}
	return fmt.Errorf("nullTime: cannot parse timestamp %q", s)
}
