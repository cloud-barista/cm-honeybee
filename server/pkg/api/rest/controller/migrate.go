package controller

import (
	"strconv"

	"github.com/cloud-barista/cm-honeybee/server/db"
	"github.com/cloud-barista/cm-honeybee/server/lib/openbao"
	"github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
)

// MigratePlaintextSecretsToOpenBao moves SSH secrets that are still stored in the
// SQLite DB into OpenBao, then blanks the DB columns. Connections created before
// the OpenBao backend was introduced kept their password/private_key directly in
// the DB (in plaintext); this lets those rows self-heal on the next restart once
// OpenBao is configured.
//
// It runs at startup and is a no-op when OpenBao is disabled or when no DB row
// still carries a secret, so it is safe (and idempotent) to call every boot.
func MigratePlaintextSecretsToOpenBao() {
	if !openbao.Enabled() {
		return
	}

	var conns []model.ConnectionInfo
	// Rows that still carry a secret in the DB. "-" is the sentinel for "no key".
	err := db.DB.Where(
		"(password IS NOT NULL AND password <> '') OR "+
			"(private_key IS NOT NULL AND private_key <> '' AND private_key <> '-')",
	).Find(&conns).Error
	if err != nil {
		logger.Println(logger.ERROR, true, "OpenBao migration: failed to scan connections: "+err.Error())
		return
	}
	if len(conns) == 0 {
		return
	}

	migrated := 0
	for i := range conns {
		ci := &conns[i]

		data := map[string]string{"password": ci.Password, "private_key": ci.PrivateKey}
		if err := openbao.Put(sshSecretPath(ci.ID), data); err != nil {
			logger.Println(logger.ERROR, true, "OpenBao migration: failed to store secrets for connection "+ci.ID+": "+err.Error())
			continue
		}

		// Clear the plaintext from the DB now that OpenBao holds it.
		if err := db.DB.Model(&model.ConnectionInfo{}).Where("id = ?", ci.ID).
			Updates(map[string]interface{}{"password": "", "private_key": ""}).Error; err != nil {
			logger.Println(logger.ERROR, true, "OpenBao migration: stored secrets but failed to clear DB row "+ci.ID+": "+err.Error())
			continue
		}
		migrated++
	}

	logger.Println(logger.INFO, true, "OpenBao migration: moved SSH secrets for "+
		strconv.Itoa(migrated)+"/"+strconv.Itoa(len(conns))+" connection(s) from DB to OpenBao")
}
