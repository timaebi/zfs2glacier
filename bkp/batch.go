package bkp

import (
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws"
	"errors"
	"github.com/aws/aws-sdk-go/service/glacier/glacieriface"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strconv"
	"io"
)

type Batch struct {
	filter         string
	filesystems    []Filesystem
	initialized    bool
	existingVaults []*glacier.DescribeVaultOutput
	glacier        glacieriface.GlacierAPI
}

// NewBatch creates a new batch
func NewBatch(filter string) (*Batch, error) {
	g, err := SetupGlacierClient()
	if err != nil {
		return nil, err
	}
	return &Batch{filter: filter, glacier: g}, nil
}

func SetupGlacierClient() (glacieriface.GlacierAPI, error) {
	// Setup AWS client
	s, err := session.NewSessionWithOptions(session.Options{
		SharedConfigFiles: []string{"/etc/aws.conf"}, // TODO make this configurable with
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}
	return glacier.New(s), nil
}

// Init prepares the Batch for execution with Run
// It searches for ZFS filesystems which are tagged for backup and whose next backup is due.
// Initialize aws client
// It also checks which vaults already exist on aws glacier
func (b *Batch) Init() error {
	// search for ZFS filesystems
	d, err := ListZFSFilesystems(b.filter)
	if err != nil {
		return err
	}
	b.filesystems = d

	// List existing vaults
	v, err := b.glacier.ListVaults(&glacier.ListVaultsInput{AccountId: aws.String("-")})
	if err != nil {
		return err
	}
	b.existingVaults = v.VaultList

	b.initialized = true
	return nil
}

// Run does the acutual backup on aws glacier
// 1. create a snapshot of each ZFSFilesystem to backup
// 2. create vaults for volumes without an existing vault
// 3. create diff to previous snapshot
// 4. upload one snapshot after the other
func (b *Batch) Run() error {
	if !b.initialized {
		return errors.New("batch needs to be initialized before run")
	}

	for _, fs := range b.filesystems {
		if fs.IsBackupEnabled() {
			forceFull := false
			vn := fs.GetVaultName()
			if !b.vaultExists(vn) {
				// create vault and force full backup
				_, err := b.glacier.CreateVault(&glacier.CreateVaultInput{
					AccountId: aws.String("-"),
					VaultName: aws.String(vn),
				})
				if err != nil {
					return err
				}
				log.WithField("vault", vn).Info("vault created")
				forceFull = true
			}
			due := fs.IsDue()
			if due || forceFull {
				log.WithField("vault", vn).Info("starting backup")
				backup := fs.Backup(forceFull)
				if err := b.upload(vn, backup); err != nil {
					return err
				}
				log.WithField("vault", vn).Info("backup finished")
			}
		}
	}
	return nil
}

func (b *Batch) upload(vault string, bkp Backup) error {
	o, err := b.glacier.InitiateMultipartUpload(&glacier.InitiateMultipartUploadInput{
		AccountId:          aws.String("-"),
		ArchiveDescription: aws.String(""),
		PartSize:           aws.String(strconv.Itoa(bkp.GetPartSize())),
		VaultName:          &vault,
	})
	if err != nil {
		return err
	}
	pos := int64(0)
	hashes := make([][]byte, 0, 100)
	for bkp.HasNextPart() {
		p, h := bkp.NextPart()
		hashes = append(hashes, h)
		l, err := p.Seek(0, io.SeekEnd)
		if err != nil {
			return err
		}
		_, err = p.Seek(0, io.SeekStart)
		if err != nil {
			return err
		}
		r := fmt.Sprintf("bytes %d-%d/*", pos, pos+l-1)
		pos = pos + l

		treeHash := fmt.Sprintf("%x", h)

		log.WithField("range", r).Debug("uploading range")
		uo, err := b.glacier.UploadMultipartPart(&glacier.UploadMultipartPartInput{
			AccountId: aws.String("-"),
			Body:      p,
			Checksum:  &treeHash,
			Range:     &r,
			UploadId:  o.UploadId,
			VaultName: &vault,
		})
		fmt.Print(uo.String())
		if err != nil {
			return err
		}
	}
	fullHash := fmt.Sprintf("%x", glacier.ComputeTreeHash(hashes))
	cu, err := b.glacier.CompleteMultipartUpload(&glacier.CompleteMultipartUploadInput{
		AccountId:   aws.String("-"),
		ArchiveSize: aws.String(strconv.FormatInt(pos, 10)),
		Checksum:    &fullHash,
		UploadId:    o.UploadId,
		VaultName:   &vault,
	})
	if err != nil {
		return err
	}
	fmt.Print(cu.String())
	bkp.MarkSuccessful(*cu.ArchiveId)
	return nil
}

func (b *Batch) vaultExists(name string) bool {
	for _, v := range b.existingVaults {
		if *v.VaultName == name {
			return true
		}
	}
	return false
}

func (b *Batch) Print() {
	const fmtStr = "%-50s | %-30s | %-30s | %-20s | %-20s\n"
	fmt.Printf(fmtStr, "Name", "Last full bkp", "Last incr bkp", "Incremental interval", "Vault archives")
	fmt.Println("------------------------------------------------------------------------------------------------------------------------------------------------------")
	for _, fs := range b.filesystems {
		if !fs.IsBackupEnabled() {
			continue
		}
		ds := fs.(*ZFSFilesystem)
		np := ds.dataset.GetNativeProperties()

		name := np.Name

		lfb := ds.getLastFullBackup()
		lastFullBackup := "-"
		if lfb != nil {
			lastFullBackup = lfb.GetNativeProperties().Creation.String()
		}

		lib := ds.getLastIncrementalBackup()
		lastIncrBackup := "-"
		if lib != nil {
			lastIncrBackup = lib.GetNativeProperties().Creation.String()
		}

		incrInterval := ds.getIncrementalInterval()

		archives := "-"
		vn := fs.GetVaultName()
		for _, v := range b.existingVaults {
			if *v.VaultName == vn {
				archives = fmt.Sprintf("%3.1fGB (%d)", float64(*v.SizeInBytes)/1e9, *v.NumberOfArchives)
			}
		}
		fmt.Printf(fmtStr, name, lastFullBackup, lastIncrBackup, incrInterval, archives)
	}
}
