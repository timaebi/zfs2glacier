// Code generated by mockery v1.0.0
package bkp

import mock "github.com/stretchr/testify/mock"
import zfsiface "github.com/timaebi/go-zfs/zfsiface"

// zfsAPI is an autogenerated mock type for the zfsAPI type
type zfsAPIMock struct {
	mock.Mock
}

// filesystems provides a mock function with given fields: filter
func (_m *zfsAPIMock) filesystems(filter string) ([]zfsiface.Dataset, error) {
	ret := _m.Called(filter)

	var r0 []zfsiface.Dataset
	if rf, ok := ret.Get(0).(func(string) []zfsiface.Dataset); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]zfsiface.Dataset)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}