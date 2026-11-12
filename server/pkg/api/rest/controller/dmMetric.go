package controller

import (
	datamig "github.com/cloud-barista/cm-centipede/data-mig"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

// The mappers below turn a request's metric selection into data-mig's. They
// copy pointers, never values: a field the caller left out stays nil so that
// data-mig applies its own default. Writing false there instead would quietly
// turn off figures that default to on (folder_mod_time, folder_perms).

// fsMetric maps a file system metric request. nil → nil (library defaults).
func fsMetric(req *model.FSMetricReq) *datamig.FilesystemMetric {
	if req == nil {
		return nil
	}

	return &datamig.FilesystemMetric{
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
func osMetric(req *model.OSMetricReq) *datamig.ObjectStorageMetric {
	if req == nil {
		return nil
	}

	return &datamig.ObjectStorageMetric{
		TotalSize:         req.TotalSize,
		ObjectCount:       req.ObjectCount,
		ExtensionCount:    req.ExtensionCount,
		PrefixModTime:     req.PrefixModTime,
		PrefixObjectCount: req.PrefixObjectCount,
		PrefixObjectSize:  req.PrefixObjectSize,
	}
}

// dbMetric maps a DBMS metric request onto the one engine dbType names, dropping
// the fields that engine does not have. nil, or a dbType data-mig does not
// know, → nil (library defaults); the connection handler rejects unknown types
// at registration, so the latter is only a fallback.
func dbMetric(dbType string, req *model.DBMetricReq) *datamig.DBMSMetricOption {
	if req == nil {
		return nil
	}

	switch dbType {
	case datamig.DBMSTypeMySQL:
		return &datamig.DBMSMetricOption{MySQL: &datamig.MySQLMetric{
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
	case datamig.DBMSTypeMariaDB:
		// Same fields as MySQL, but data-mig keeps a type of its own.
		return &datamig.DBMSMetricOption{MariaDB: &datamig.MariaDBMetric{
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
	case datamig.DBMSTypePostgreSQL:
		return &datamig.DBMSMetricOption{PostgreSQL: &datamig.PostgreSQLMetric{
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
	case datamig.DBMSTypeMongoDB:
		return &datamig.DBMSMetricOption{MongoDB: &datamig.MongoDBMetric{
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
