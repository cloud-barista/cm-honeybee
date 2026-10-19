package model

import (
	"time"

	sourcemodel "github.com/cloud-barista/cm-centipede/dmdl/source-model"
)

// SavedDBInfo statuses. Constants for the same reason as SavedFSInfoStatus*.
const (
	SavedDBInfoStatusSuccess = "success"
	// SavedDBInfoStatusPartial means some databases were collected and others
	// were skipped (for example for lack of privileges); see DBInspectResult.
	SavedDBInfoStatusPartial = "partial"
	SavedDBInfoStatusFailed  = "failed"
	// SavedDBInfoStatusStale marks a result collected for a database or schema
	// selection the connection no longer has.
	SavedDBInfoStatusStale = "stale"
)

// ImportDBReq selects how much to collect. Where to connect is all on the
// connection.
type ImportDBReq struct {
	Metric *DBMetricReq `json:"metric,omitempty"`
}

// DBMetricReq picks the optional DBMS figures. Only the fields the
// connection's db_type has are used; the rest are ignored. An omitted field
// keeps the library default (all off, sample_size 100).
type DBMetricReq struct {
	// All engines.
	RowCountExact *bool `json:"row_count_exact,omitempty"`
	Columns       *bool `json:"columns,omitempty"`
	Indexes       *bool `json:"indexes,omitempty"`
	Views         *bool `json:"views,omitempty"`

	// MySQL, MariaDB and PostgreSQL.
	ForeignKeys *bool `json:"foreign_keys,omitempty"`
	Functions   *bool `json:"functions,omitempty"`
	Procedures  *bool `json:"procedures,omitempty"`
	Triggers    *bool `json:"triggers,omitempty"`

	// MySQL and MariaDB.
	Events *bool `json:"events,omitempty"`

	// PostgreSQL.
	MaterializedViews *bool `json:"materialized_views,omitempty"`
	Sequences         *bool `json:"sequences,omitempty"`
	Types             *bool `json:"types,omitempty"`
	Extensions        *bool `json:"extensions,omitempty"`
	Rules             *bool `json:"rules,omitempty"`

	// MongoDB.
	Validators *bool `json:"validators,omitempty"`
	SampleSize *int  `json:"sample_size,omitempty"`
}

// SavedDBInfo is the last DBMS collection of a connection, DBData holding a
// DBInspectResult.
type SavedDBInfo struct {
	ConnectionID string    `gorm:"primaryKey" json:"connection_id" validate:"required"`
	DBData       string    `gorm:"column:db_data" json:"db_data" validate:"required"`
	Status       string    `gorm:"column:status" json:"status"`
	SavedTime    time.Time `gorm:"column:saved_time" json:"saved_time"`
}

// DBInspectResult is what one connection's DBMS collection stored. A connection
// names a server, so Databases is a list even when db_name selected a single
// database: one shape for both.
type DBInspectResult struct {
	Databases []sourcemodel.DBMSInfo `json:"databases"`
	// Skipped lists the databases that could not be collected when the whole
	// server was enumerated.
	Skipped []DBSkippedEntry `json:"skipped,omitempty"`
}

// DBSkippedEntry is one database left out of a collection, with why. The tags
// match transx-ex's DBMSInspectError.
type DBSkippedEntry struct {
	Database string `json:"database"`
	Error    string `json:"error"`
}

type DBInfoList struct {
	Servers []DBInspectResult `json:"servers" validate:"required"`
}
