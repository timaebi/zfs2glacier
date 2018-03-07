package bkp

import (
	"testing"
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/timaebi/go-zfs/zfsiface"
	"github.com/stretchr/testify/mock"
)

func TestZfsBackup_NextPart(t *testing.T) {
	//data:= make([]byte, 10)
	const chunckLength = 1024 * 1024 * 10

	data := []byte{1, 2, 3}
	b := zfsBackup{zfsReader: bytes.NewBuffer(data), data: make([]byte, chunckLength), hasNext: true}
	assert.True(t, b.HasNextPart())
	p, h := b.NextPart()
	readBuf := make([]byte, b.GetPartSize())
	n, err := p.Read(readBuf)
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, data, readBuf[:n])
	assert.NotEmpty(t, h)
	assert.False(t, b.HasNextPart())
	assert.Panics(t, func() {
		b.NextPart()
	})
}

func TestZfsBackup_MarkSuccessful(t *testing.T) {
	d := &Dataset{}
	d.On("SetProperty", glacierArchiveID, "received-archive-id").Return(nil).Once()
	d.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "test/fs@glacier-tmp"}).Once()
	d.On("Rename", "test/fs@glacier-full", false, false).Return(&Dataset{}, nil)
	b := zfsBackup{dataset: d}
	err := b.MarkSuccessful("received-archive-id")
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, d)

	base := &Dataset{}
	base.On("Destroy", zfsiface.DestroyDefault).Return(nil).Once()
	base.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "test/fs@glacier-incremental"}).Once()
	d = &Dataset{}
	d.On("SetProperty", glacierArchiveID, "received-archive-id").Return(nil).Once()
	d.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "test/fs@glacier-tmp"}).Once()
	d.On("Rename", "test/fs@glacier-incremental", false, false).Return(&Dataset{}, nil)
	b = zfsBackup{dataset: d, base: base}
	err = b.MarkSuccessful("received-archive-id")
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, d)
	mock.AssertExpectationsForObjects(t, base)

	base = &Dataset{}
	base.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "test/fs@glacier-full"}).Once()
	d = &Dataset{}
	d.On("SetProperty", glacierArchiveID, "received-archive-id").Return(nil).Once()
	d.On("GetNativeProperties").Return(&zfsiface.NativeProperties{Name: "test/fs@glacier-tmp"}).Once()
	d.On("Rename", "test/fs@glacier-incremental", false, false).Return(&Dataset{}, nil)
	b = zfsBackup{dataset: d, base: base}
	err = b.MarkSuccessful("received-archive-id")
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, d)
	mock.AssertExpectationsForObjects(t, base)
}
