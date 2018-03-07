package bkp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/aws/aws-sdk-go/service/glacier"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/timaebi/go-zfs/zfsiface"
	"errors"
)

//func TestNewBatch(t *testing.T) {
//	b := NewBatch("tank/test")
//	assert.Equal(t, "tank/test", b.filter)
//}

func TestBatch_Init(t *testing.T) {
	api := &GlacierAPI{}
	existingVaults := []*glacier.DescribeVaultOutput{
		{CreationDate: aws.String("2012-03-20T17:03:43.221Z"), VaultName: aws.String("tank_testit")},
	}
	api.On("ListVaults", mock.AnythingOfType("*glacier.ListVaultsInput")).
		Return(&glacier.ListVaultsOutput{VaultList: existingVaults}, nil)
	b := &Batch{filter: "tank/test", glacier: api}
	m := &zfsAPIMock{}
	m.On("filesystems", "tank/test").
		Return([]zfsiface.Dataset{}, nil)
	defaultAPI = m
	err := b.Init()
	assert.NoError(t, err)
	assert.True(t, b.initialized)
	assert.Equal(t, existingVaults, b.existingVaults)

	api = &GlacierAPI{}
	api.On("ListVaults", mock.AnythingOfType("*glacier.ListVaultsInput")).
		Return(&glacier.ListVaultsOutput{VaultList: existingVaults}, nil)
	b = &Batch{filter: "tank/test", glacier: api}
	m = &zfsAPIMock{}
	m.On("filesystems", "tank/test").
		Return(nil, errors.New("Simulated error"))
	defaultAPI = m
	err = b.Init()
	assert.Error(t, err)
	assert.False(t, b.initialized)

	api = &GlacierAPI{}
	api.On("ListVaults", mock.AnythingOfType("*glacier.ListVaultsInput")).
		Return(nil, errors.New("Simulated error"))
	b = &Batch{filter: "tank/test", glacier: api}
	m = &zfsAPIMock{}
	m.On("filesystems", "tank/test").
		Return([]zfsiface.Dataset{}, nil)
	defaultAPI = m
	err = b.Init()
	assert.Error(t, err)
	assert.False(t, b.initialized)
}

func TestBatch_Run(t *testing.T) {
	api := &GlacierAPI{}
	b := &Batch{filter: "tank/test", glacier: api}
	err := b.Run()
	assert.Error(t, err)
}
