package controller

import (
	transxex "github.com/cloud-barista/cm-centipede/transx-ex"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// The mappers below turn a request's metric selection into transx-ex's. They
// copy pointers, never values: a field the caller left out stays nil so that
// transx-ex applies its own default. Writing false there instead would quietly
// turn off figures that default to on (folder_mod_time, folder_perms).

// fsMetric maps a file system metric request. nil → nil (library defaults).
func fsMetric(req *model.FSMetricReq) *transxex.FilesystemMetric {
	if req == nil {
		return nil
	}

	return &transxex.FilesystemMetric{
		TotalSize:       req.TotalSize,
		FileCount:       req.FileCount,
		ExtensionCount:  req.ExtensionCount,
		FolderModTime:   req.FolderModTime,
		FolderPerms:     req.FolderPerms,
		FolderFileCount: req.FolderFileCount,
		FolderFileSize:  req.FolderFileSize,
	}
}

// osMetric maps an object storage metric request. nil → nil (library defaults).
func osMetric(req *model.OSMetricReq) *transxex.ObjectStorageMetric {
	if req == nil {
		return nil
	}

	return &transxex.ObjectStorageMetric{
		TotalSize:         req.TotalSize,
		ObjectCount:       req.ObjectCount,
		ExtensionCount:    req.ExtensionCount,
		PrefixModTime:     req.PrefixModTime,
		PrefixObjectCount: req.PrefixObjectCount,
		PrefixObjectSize:  req.PrefixObjectSize,
	}
}

// dbMetric maps a DBMS metric request onto the one engine dbType names, dropping
// the fields that engine does not have. nil, or a dbType transx-ex does not
// know, → nil (library defaults); the connection handler rejects unknown types
// at registration, so the latter is only a fallback.
func dbMetric(dbType string, req *model.DBMetricReq) *transxex.DBMSMetricOption {
	if req == nil {
		return nil
	}

	switch dbType {
	case transxex.DBMSTypeMySQL:
		return &transxex.DBMSMetricOption{MySQL: &transxex.MySQLMetric{
			RowCountExact: req.RowCountExact,
			Columns:       req.Columns,
			Indexes:       req.Indexes,
			ForeignKeys:   req.ForeignKeys,
			Views:         req.Views,
			Functions:     req.Functions,
			Procedures:    req.Procedures,
			Triggers:      req.Triggers,
			Events:        req.Events,
		}}
	case transxex.DBMSTypeMariaDB:
		// Same fields as MySQL, but transx-ex keeps a type of its own.
		return &transxex.DBMSMetricOption{MariaDB: &transxex.MariaDBMetric{
			RowCountExact: req.RowCountExact,
			Columns:       req.Columns,
			Indexes:       req.Indexes,
			ForeignKeys:   req.ForeignKeys,
			Views:         req.Views,
			Functions:     req.Functions,
			Procedures:    req.Procedures,
			Triggers:      req.Triggers,
			Events:        req.Events,
		}}
	case transxex.DBMSTypePostgreSQL:
		return &transxex.DBMSMetricOption{PostgreSQL: &transxex.PostgreSQLMetric{
			RowCountExact:     req.RowCountExact,
			Columns:           req.Columns,
			Indexes:           req.Indexes,
			ForeignKeys:       req.ForeignKeys,
			Views:             req.Views,
			MaterializedViews: req.MaterializedViews,
			Functions:         req.Functions,
			Procedures:        req.Procedures,
			Triggers:          req.Triggers,
			Sequences:         req.Sequences,
			Types:             req.Types,
			Extensions:        req.Extensions,
			Rules:             req.Rules,
		}}
	case transxex.DBMSTypeMongoDB:
		return &transxex.DBMSMetricOption{MongoDB: &transxex.MongoDBMetric{
			RowCountExact: req.RowCountExact,
			Columns:       req.Columns,
			Indexes:       req.Indexes,
			Views:         req.Views,
			Validators:    req.Validators,
			SampleSize:    req.SampleSize,
		}}
	}

	return nil
}
