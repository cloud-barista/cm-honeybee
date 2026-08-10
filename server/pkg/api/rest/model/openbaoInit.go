package model

// OpenBaoInit stores cm-honeybee's self-managed OpenBao unseal material (the
// unseal key + root token generated at 'operator init'). Both values are
// RSA-encrypted at rest with honeybee's key (honeybee.pub to write,
// honeybee.key to read), so the DB never holds them in plaintext. A single row
// is kept (ID = 1).
type OpenBaoInit struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	UnsealKeyEnc string `gorm:"column:unseal_key_enc" json:"-"`
	RootTokenEnc string `gorm:"column:root_token_enc" json:"-"`
}
