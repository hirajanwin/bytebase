package api

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap/zapcore"
)

const (
	// BackupRetentionPeriodUnset is the unset value of a backup retention period.
	BackupRetentionPeriodUnset = 0
)

// BackupStatus is the status of a backup.
type BackupStatus string

const (
	// BackupStatusPendingCreate is the status for PENDING_CREATE.
	BackupStatusPendingCreate BackupStatus = "PENDING_CREATE"
	// BackupStatusDone is the status for DONE.
	BackupStatusDone BackupStatus = "DONE"
	// BackupStatusFailed is the status for FAILED.
	BackupStatusFailed BackupStatus = "FAILED"
)

// BackupType is the type of a backup.
type BackupType string

const (
	// BackupTypeAutomatic is the type for automatic backup.
	BackupTypeAutomatic BackupType = "AUTOMATIC"
	// BackupTypePITR is the type of backup taken at PITR cutover stage.
	BackupTypePITR BackupType = "PITR"
	// BackupTypeManual is the type for manual backup.
	BackupTypeManual BackupType = "MANUAL"
)

// BackupStorageBackend is the storage backend of a backup.
type BackupStorageBackend string

const (
	// BackupStorageBackendLocal is the local storage backend for a backup.
	BackupStorageBackendLocal BackupStorageBackend = "LOCAL"
	// BackupStorageBackendS3 is the AWS S3 storage backend for a backup. Not used yet.
	BackupStorageBackendS3 BackupStorageBackend = "S3"
	// BackupStorageBackendGCS is the Google Cloud Storage (GCS) storage backend for a backup. Not used yet.
	BackupStorageBackendGCS BackupStorageBackend = "GCS"
	// BackupStorageBackendOSS is the AliCloud Object Storage Service (OSS) storage backend for a backup. Not used yet.
	BackupStorageBackendOSS BackupStorageBackend = "OSS"
)

// BinlogInfo is the binlog coordination for MySQL.
type BinlogInfo struct {
	FileName string `json:"fileName"`
	Position int64  `json:"position"`
}

// IsEmpty return true if the BinlogInfo is empty.
func (b BinlogInfo) IsEmpty() bool {
	return b == BinlogInfo{}
}

// BackupPayload contains backup related database specific info, it differs for different database types.
// It is encoded in JSON and stored in the backup table.
type BackupPayload struct {
	// MySQL related fields
	// BinlogInfo is recorded when taking the backup.
	// It is recorded within the same transaction as the dump so that the binlog position is consistent with the dump.
	// Please refer to https://github.com/bytebase/bytebase/blob/main/docs/design/pitr-mysql.md#full-backup for details.
	BinlogInfo BinlogInfo `json:"binlogInfo"`
}

// Backup is the API message for a backup.
type Backup struct {
	ID int `jsonapi:"primary,backup"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name           string               `jsonapi:"attr,name"`
	Status         BackupStatus         `jsonapi:"attr,status"`
	Type           BackupType           `jsonapi:"attr,type"`
	StorageBackend BackupStorageBackend `jsonapi:"attr,storageBackend"`
	// Upon taking the database backup, we will also record the current migration history version if exists.
	// And when restoring the backup, we will record this in the migration history.
	MigrationHistoryVersion string `jsonapi:"attr,migrationHistoryVersion"`
	Path                    string `jsonapi:"attr,path"`
	Comment                 string `jsonapi:"attr,comment"`
	// Payload contains data such as binlog position info which will not be created at first.
	// It is filled when the backup task executor takes database backups.
	Payload BackupPayload `jsonapi:"attr,payload"`
}

// ZapBackupArray is a helper to format zap.Array.
type ZapBackupArray []*Backup

// MarshalLogArray implements the zapcore.ArrayMarshaler interface.
func (backups ZapBackupArray) MarshalLogArray(arr zapcore.ArrayEncoder) error {
	for _, backup := range backups {
		payload, err := json.Marshal(backup.Payload)
		if err != nil {
			return err
		}
		arr.AppendString(fmt.Sprintf("{name:%s, id:%d, payload:%s}", backup.Name, backup.ID, payload))
	}
	return nil
}

// BackupCreate is the API message for creating a backup.
type BackupCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	DatabaseID int `jsonapi:"attr,databaseId"`

	// Domain specific fields
	Name                    string               `jsonapi:"attr,name"`
	Type                    BackupType           `jsonapi:"attr,type"`
	StorageBackend          BackupStorageBackend `jsonapi:"attr,storageBackend"`
	MigrationHistoryVersion string
	Path                    string
}

// BackupFind is the API message for finding backups.
type BackupFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
	Name   *string
	Status *BackupStatus
}

func (find *BackupFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// BackupPatch is the API message for patching a backup.
type BackupPatch struct {
	ID int

	// Standard fields
	RowStatus *RowStatus
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  string
	Comment string
	Payload string
}

// BackupSetting is the backup setting for a database.
type BackupSetting struct {
	ID int `jsonapi:"primary,backupSetting"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	DatabaseID int `jsonapi:"attr,databaseId"`
	// Do not return this to the client since the client always has the database context and fetching the
	// database object and all its own related objects is a bit expensive.
	Database *Database

	// Domain specific fields
	Enabled bool `jsonapi:"attr,enabled"`
	// Schedule related fields
	Hour      int `jsonapi:"attr,hour"`
	DayOfWeek int `jsonapi:"attr,dayOfWeek"`
	// RetentionPeriodTs is the period that backup data is kept for the database.
	// 0 means unset and we do not delete data.
	RetentionPeriodTs int `jsonapi:"attr,retentionPeriodTs"`
	// HookURL is the callback url to be requested (using HTTP GET) after a successful backup.
	HookURL string `jsonapi:"attr,hookUrl"`
}

// BackupSettingFind is the message to get a backup settings.
type BackupSettingFind struct {
	ID *int

	// Related fields
	DatabaseID *int

	// Domain specific fields
	InstanceID *int
}

// BackupSettingUpsert is the message to upsert a backup settings.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type BackupSettingUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorID is the ID of the creator.
	UpdaterID int

	// Related fields
	DatabaseID    int `jsonapi:"attr,databaseId"`
	EnvironmentID int

	// Domain specific fields
	Enabled           bool   `jsonapi:"attr,enabled"`
	Hour              int    `jsonapi:"attr,hour"`
	DayOfWeek         int    `jsonapi:"attr,dayOfWeek"`
	RetentionPeriodTs int    `jsonapi:"attr,retentionPeriodTs"`
	HookURL           string `jsonapi:"attr,hookUrl"`
}

// BackupSettingsMatch is the message to find backup settings matching the conditions.
type BackupSettingsMatch struct {
	Hour      int
	DayOfWeek int
}
