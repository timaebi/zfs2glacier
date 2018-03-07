package bkp

import (
	"github.com/timaebi/go-zfs"
	"github.com/timaebi/go-zfs/zfsiface"
	"strings"
	"regexp"
	"time"
	"strconv"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type backupType int

const (
	full        backupType = iota
	incremental
	none
)
const BackupEnabled = "ch.floor4:backup_enabled"
const IncrementalInterval = "ch.floor4:incremental_interval"

// A Filesystem provides all information to decide if a backup should be done
type Filesystem interface {
	// IsBackupEnabled returns true if the backup it should be backed up on a regular basis
	IsBackupEnabled() bool
	// GetVaultName transforms the filesystem name into a valid aws vault name
	// It replaces all non -a-zA-Z0-9 characters with underscores
	GetVaultName() string
	// IsDue returns true if it is time for a next backup
	IsDue() bool
	// Backup returns a Backup which can be started. It will then write the backup to the given writer.
	// Depending on the backup history it decides if a full or an incremental backup should be done.
	Backup(forceFull bool) Backup
}

// ZFSFilesystem extends the go-zfs ZFSFilesystem with properties needed for
type ZFSFilesystem struct {
	dataset zfsiface.Dataset
}

// GetVaultName transforms the filesystem name into a valid aws vault name
// It replaces all non -a-zA-Z0-9 characters with underscores
func (fs *ZFSFilesystem) GetVaultName() string {
	// replace every underscore with two underscores
	v := strings.Replace(fs.dataset.GetNativeProperties().Name, "_", "__", -1)
	// replace every illegal character
	re := regexp.MustCompile("[^-a-zA-Z0-9_]")
	return re.ReplaceAllString(v, "_")
}

// IsDue returns true if it is time for a next backup
func (fs *ZFSFilesystem) IsDue() bool {
	return fs.nextBackupType() != none
}

func (fs *ZFSFilesystem) getIncrementalInterval() time.Duration {
	str, _, err := fs.dataset.GetProperty(IncrementalInterval)
	if err != nil {
		str = "2592000" // 30 days
	}
	num, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		num = 2592000 // 30 days
	}
	return time.Duration(num) * time.Second
}

// Backup returns a Backup which can be started. It will then write the backup to the given writer.
// Depending on the backup history it decides if a full or an incremental backup should be done.
func (fs *ZFSFilesystem) Backup(forceFull bool) Backup {
	nextType := fs.nextBackupType()
	if nextType == none {
		return nil
	}

	existingSnap := fs.findSnapshotWithName("glacier-tmp")
	base := fs.findBaseSnapshot()
	if existingSnap != nil {
		if !forceFull && nextType == incremental {
			return newBackup(existingSnap, base)
		} else if nextType == full {
			return newBackup(existingSnap, nil)
		}
	}
	snap, err := fs.dataset.Snapshot("glacier-tmp", false)
	if err != nil {
		panic(err)
	}
	if !forceFull && nextType == incremental {
		return newBackup(snap, base)
	} else if nextType == full {
		return newBackup(snap, nil)
	}

	return nil
}

// IsBackupEnabled returns true if the backup it should be backed up on a regular basis
func (fs *ZFSFilesystem) IsBackupEnabled() bool {
	enabled, ps, err := fs.dataset.GetProperty(BackupEnabled)
	if err != nil || ps != zfsiface.Local {
		return false
	}
	return cases.Lower(language.English).String(enabled) == "true" || enabled == "1"
}

func (fs *ZFSFilesystem) getLastFullBackup() zfsiface.Dataset {
	return fs.findSnapshotWithName("glacier-full")
}

func (fs *ZFSFilesystem) getLastIncrementalBackup() zfsiface.Dataset {
	return fs.findSnapshotWithName("glacier-incremental")
}

func (fs *ZFSFilesystem) findSnapshotWithName(name string) zfsiface.Dataset {
	snaps, err := fs.dataset.Snapshots()
	if err != nil {
		panic(err)
	}
	for _, snap := range snaps {
		n := snap.GetNativeProperties().Name
		parts := strings.Split(n, "@")
		if len(parts) != 2 {
			panic("unexpected snapshot name format " + n)
		}
		if parts[1] == name {
			return snap
		}
	}
	return nil
}

func (fs *ZFSFilesystem) nextBackupType() backupType {
	lfb := fs.getLastFullBackup()
	if lfb == nil {
		return full
	}
	lib := fs.getLastIncrementalBackup()
	if lib == nil {
		if time.Since(lfb.GetNativeProperties().Creation) > fs.getIncrementalInterval() {
			return incremental
		}
	} else if time.Since(lib.GetNativeProperties().Creation) > fs.getIncrementalInterval() {
		return incremental
	}
	return none
}

func (fs *ZFSFilesystem) findBaseSnapshot() zfsiface.Dataset {
	incremental := fs.getLastIncrementalBackup()
	if incremental != nil {
		return incremental
	}
	full := fs.getLastFullBackup()
	if full != nil {
		return full
	}
	return nil
}

type zfsAPI interface {
	filesystems(filter string) ([]zfsiface.Dataset, error)
}

type api struct{}

func (api *api) filesystems(filter string) ([]zfsiface.Dataset, error) {
	return zfs.Filesystems(filter)
}

var defaultAPI zfsAPI = &api{}

// ListZFSFilesystems returns a list of all zfs filesystems under the path given by filter
func ListZFSFilesystems(filter string) ([]Filesystem, error) {
	datasets, err := defaultAPI.filesystems(filter)
	if err != nil {
		return nil, err
	}
	fsList := make([]Filesystem, len(datasets))
	for i, ds := range datasets {
		fsList[i] = &ZFSFilesystem{ds}
	}
	return fsList, nil
}
