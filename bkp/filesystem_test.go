package bkp

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/timaebi/go-zfs/zfsiface"
	"time"
	"errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestZFSFilesystem_VaultName(t *testing.T) {
	m := &Dataset{}
	m.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "tank/test/volume"})
	d := ZFSFilesystem{m}
	assert.Equal(t, "tank_test_volume", d.GetVaultName())

	m = &Dataset{}
	m.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "tank/test_volume/with_underscores"})
	d = ZFSFilesystem{m}
	assert.Equal(t, "tank_test__volume_with__underscores", d.GetVaultName())
}

func TestZFSFilesystem_IsBackupEnabled(t *testing.T) {
	m := &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("1", zfsiface.Local, nil)
	d := ZFSFilesystem{m}
	assert.True(t, d.IsBackupEnabled())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("true", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.True(t, d.IsBackupEnabled())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("tRuE", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.True(t, d.IsBackupEnabled())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("false", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.False(t, d.IsBackupEnabled())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("0", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.False(t, d.IsBackupEnabled())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("fooBAR", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.False(t, d.IsBackupEnabled())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:backup_enabled").
		Return("true", zfsiface.Inherited, nil)
	d = ZFSFilesystem{m}
	assert.False(t, d.IsBackupEnabled())
}

func TestZFSFilesystem_GetIncrementalInterval(t *testing.T) {
	const defaultIncremental = 24 * 30 * time.Hour

	m := &Dataset{}
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("3600", zfsiface.Local, nil)
	d := ZFSFilesystem{m}
	assert.Equal(t, time.Hour, d.getIncrementalInterval())

	m = &Dataset{}
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("abc", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.Equal(t, defaultIncremental, d.getIncrementalInterval())

	m = &Dataset{}
	m.On("GetProperty", mock.AnythingOfType("string")).
		Return("", zfsiface.Local, errors.New("Simulated error"))
	d = ZFSFilesystem{m}
	assert.Equal(t, defaultIncremental, d.getIncrementalInterval())
}

func TestZFSFilesystem_IsDue(t *testing.T) {
	full2HoursAgo := &Dataset{}
	full2HoursAgo.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{
		Creation: time.Now().Add(-2 * time.Hour),
		Name:     "tank/test@glacier-full",
	})

	incremental1HourAgo := &Dataset{}
	incremental1HourAgo.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{
		Creation: time.Now().Add(-1 * time.Hour),
		Name:     "tank/test@glacier-incremental",
	})

	// backup every half an hour, last full backup 2 hours ago, last incremental 1 hour ago
	m := &Dataset{}
	m.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{Creation: time.Now().Add(-24 * time.Hour)})
	m.On("Snapshots").
		Return([]zfsiface.Dataset{full2HoursAgo, incremental1HourAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("1800", zfsiface.Local, nil)
	d := ZFSFilesystem{m}
	assert.True(t, d.IsDue())

	// backup every 1.5 hour, last full backup 2 hours ago, last incremental 1 hour ago
	m = &Dataset{}
	m.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{Creation: time.Now().Add(-24 * time.Hour)})
	m.On("Snapshots").
		Return([]zfsiface.Dataset{full2HoursAgo, incremental1HourAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("5400", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.False(t, d.IsDue())

	// backup every 1.5 hour, last full backup 2 hours ago
	m = &Dataset{}
	m.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{Creation: time.Now().Add(-24 * time.Hour)})
	m.On("Snapshots").
		Return([]zfsiface.Dataset{full2HoursAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("5400", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.True(t, d.IsDue())

	// backup every hour hour no backup
	m = &Dataset{}
	m.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{Creation: time.Now().Add(-24 * time.Hour)})
	m.On("Snapshots").
		Return([]zfsiface.Dataset{}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("3600", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	assert.True(t, d.IsDue())
}

func TestListZFSFilesystems(t *testing.T) {
	ds := []zfsiface.Dataset{
		&Dataset{},
		&Dataset{},
	}
	m := &zfsAPIMock{}
	m.On("filesystems", "tank/test").
		Return(ds, nil)
	defaultAPI = m
	fsList, err := ListZFSFilesystems("tank/test")
	assert.NoError(t, err)
	for _, fs := range fsList {
		assert.Equal(t, ds[0], fs.(*ZFSFilesystem).dataset)
	}

	m = &zfsAPIMock{}
	m.On("filesystems", "tank/test").
		Return(nil, errors.New("Simulated error"))
	defaultAPI = m
	_, err = ListZFSFilesystems("tank/test")
	assert.Error(t, err)
}

func TestZFSFilesystem_Backup(t *testing.T) {
	full2HoursAgo := &Dataset{}
	full2HoursAgo.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{
		Creation: time.Now().Add(-2 * time.Hour),
		Name:     "tank/test@glacier-full",
	})

	incremental1HourAgo := &Dataset{}
	incremental1HourAgo.On("GetNativeProperties").
		Return(&zfsiface.NativeProperties{
		Creation: time.Now().Add(-1 * time.Hour),
		Name:     "tank/test@glacier-incremental",
	})

	// backup due no existing backup -> backup full backup should be created
	existingTmp := &Dataset{}
	existingTmp.On("SendSnapshot", mock.Anything).Return(nil).Once()
	existingTmp.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "tank/test@glacier-tmp"})
	m := &Dataset{}
	m.On("Snapshots").
		Return([]zfsiface.Dataset{}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("600", zfsiface.Local, nil)
	m.On("Snapshot", "glacier-tmp", false).
		Return(existingTmp, nil)
	d := ZFSFilesystem{m}
	b := d.Backup(false).(*zfsBackup)
	require.NotNil(t, b)
	b.NextPart()
	assert.Equal(t, "tank/test@glacier-tmp", b.dataset.GetNativeProperties().Name)

	// backup due, existing full backup -> create incremental backup with full bkp as base
	existingTmp = &Dataset{}
	existingTmp.On("SendIncrementalSnapshot", full2HoursAgo, mock.Anything).Return(nil).Once()
	existingTmp.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "tank/test@glacier-tmp"})
	m = &Dataset{}
	m.On("Snapshots").
		Return([]zfsiface.Dataset{full2HoursAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("600", zfsiface.Local, nil)
	m.On("Snapshot", "glacier-tmp", false).
		Return(existingTmp, nil)
	d = ZFSFilesystem{m}
	b = d.Backup(false).(*zfsBackup)
	b.NextPart()
	assert.Equal(t, "tank/test@glacier-tmp", b.GetDataset().GetNativeProperties().Name)
	assert.Equal(t, "tank/test@glacier-full", b.GetBaseDataset().GetNativeProperties().Name)

	// backup due, existing full backup and incremental -> create incremental backup with incremental bkp as base
	existingTmp = &Dataset{}
	existingTmp.On("SendIncrementalSnapshot", incremental1HourAgo, mock.Anything).Return(nil).Once()
	existingTmp.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "tank/test@glacier-tmp"})
	m = &Dataset{}
	m.On("Snapshots").
		Return([]zfsiface.Dataset{full2HoursAgo, incremental1HourAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("600", zfsiface.Local, nil)
	m.On("Snapshot", "glacier-tmp", false).
		Return(existingTmp, nil)
	d = ZFSFilesystem{m}
	b = d.Backup(false).(*zfsBackup)
	b.NextPart()
	assert.Equal(t, "tank/test@glacier-tmp", b.GetDataset().GetNativeProperties().Name)
	assert.Equal(t, "tank/test@glacier-incremental", b.GetBaseDataset().GetNativeProperties().Name)

	// backup not due
	m = &Dataset{}
	m.On("Snapshots").
		Return([]zfsiface.Dataset{full2HoursAgo, incremental1HourAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("10000", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	n := d.Backup(false)
	assert.Nil(t, n)

	// backup due, existing full backup and tmp snap because previous was aborted
	// -> create incremental backup with full bkp as base
	existingTmp = &Dataset{}
	existingTmp.On("SendIncrementalSnapshot", full2HoursAgo, mock.Anything).Return(nil).Once()
	existingTmp.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "tank/test@glacier-tmp"})
	m = &Dataset{}
	m.On("Snapshots").
		Return([]zfsiface.Dataset{existingTmp,full2HoursAgo}, nil)
	m.On("GetProperty", "ch.floor4:incremental_interval").
		Return("600", zfsiface.Local, nil)
	d = ZFSFilesystem{m}
	b = d.Backup(false).(*zfsBackup)
	b.NextPart()
	assert.Equal(t, "tank/test@glacier-tmp", b.GetDataset().GetNativeProperties().Name)
	assert.Equal(t, "tank/test@glacier-full", b.GetBaseDataset().GetNativeProperties().Name)
}
