package bkp

import (
	"io"
	"github.com/timaebi/go-zfs/zfsiface"
	"bytes"
	"github.com/aws/aws-sdk-go/service/glacier"
	"strings"
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

const glacierArchiveID = "ch.floor4:glacier-archive-id"

// A Backup is a backup process that when started writes the backup to a backup location
type Backup interface {
	// It can't be started again after calling MarkSuccessful
	MarkSuccessful(archiveID string) error
	GetPartSize() int
	NextPart() (io.ReadSeeker, []byte)
	HasNextPart() bool
	GetBaseDataset() zfsiface.Dataset
	GetDataset() zfsiface.Dataset
	IsIncremental() bool
	GetDescription() string
}

// Metadata contains information for a backup that is rendered as JSON
// It is saved as description field in aws
type Metadata struct {
	BaseArchiveID string `json:",omitempty"`
	IsIncremental bool
}

func newBackup(dataset, base zfsiface.Dataset) Backup {
	reader, writer := io.Pipe()
	go func() {
		var err error
		if base == nil {
			log.WithField("fs", dataset.GetNativeProperties().Name).WithField("isFull", true).
				Info("starting full backup")
			err = dataset.SendSnapshot(writer)
		} else {
			log.WithField("fs", dataset.GetNativeProperties().Name).WithField("isFull", false).
				Info("starting incremental backup")
			err = dataset.SendIncrementalSnapshot(base, writer)
		}
		if err != nil {
			err = writer.CloseWithError(err)
		} else {
			err = writer.Close()
		}
		if err != nil {
			panic("could not close writers")
		}
	}()
	b := &zfsBackup{
		data:      make([]byte, 1024*1024*128),
		hashes:    make([][]byte, 0, 128),
		zfsReader: reader,
		hasNext:   true,
		dataset:   dataset,
		base:      base,
	}
	return b
}

type zfsBackup struct {
	base      zfsiface.Dataset
	dataset   zfsiface.Dataset
	data      []byte
	hashes    [][]byte
	zfsReader io.Reader
	hasNext   bool
}

func (b *zfsBackup) GetBaseDataset() zfsiface.Dataset {
	return b.base
}

func (b *zfsBackup) GetDataset() zfsiface.Dataset {
	return b.dataset
}

func (b *zfsBackup) HasNextPart() bool {
	return b.hasNext
}

func (b *zfsBackup) GetPartSize() int {
	return len(b.data)
}

func (b *zfsBackup) IsIncremental() bool {
	return b.base != nil
}

func (b *zfsBackup) NextPart() (io.ReadSeeker, []byte) {
	if !b.hasNext {
		panic("No next chunck. Check first with HasNext")
	}
	n, err := io.ReadAtLeast(b.zfsReader, b.data, b.GetPartSize())
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		panic(err)
	}
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		b.hasNext = false
	}
	buf := bytes.NewReader(b.data[:n])
	h := glacier.ComputeHashes(buf)
	return buf, h.TreeHash
}

func (b *zfsBackup) MarkSuccessful(archiveID string) error {
	err := b.dataset.SetProperty(glacierArchiveID, archiveID)
	if err != nil {
		return err
	}
	if b.IsIncremental() {
		n := b.base.GetNativeProperties().Name
		p := strings.Split(n, "@")
		if len(p) != 2 {
			panic("unexpected snapshot name format " + n)
		}
		if p[1] == "glacier-incremental" {
			err = b.base.Destroy(zfsiface.DestroyDefault)
			if err != nil {
				return err
			}
		}
	}
	np := b.dataset.GetNativeProperties()
	p := strings.Split(np.Name, "@")
	if len(p) != 2 {
		panic("unexpected snapshot name format " + np.Name)
	}
	if b.IsIncremental() {
		b.dataset, err = b.dataset.Rename(p[0]+"@glacier-incremental", false, false)
	} else {
		b.dataset, err = b.dataset.Rename(p[0]+"@glacier-full", false, false)
	}
	if err != nil {
		return err
	}
	return nil
}

func (b *zfsBackup) GetDescription() string {
	m := &Metadata{
		IsIncremental: b.IsIncremental(),
	}
	if b.base != nil {
		bdID, _, err := b.base.GetProperty(glacierArchiveID)
		if err != nil {
			panic(err)
		}
		if bdID == "" {
			panic("No glacier archive ID found for base dataset")
		}
		m.BaseArchiveID = bdID
	}
	d, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	return string(d)
}
