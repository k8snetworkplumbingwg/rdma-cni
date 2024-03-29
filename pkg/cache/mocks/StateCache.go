// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	cache "github.com/k8snetworkplumbingwg/rdma-cni/pkg/cache"
	mock "github.com/stretchr/testify/mock"
)

// StateCache is an autogenerated mock type for the StateCache type
type StateCache struct {
	mock.Mock
}

// Delete provides a mock function with given fields: ref
func (_m *StateCache) Delete(ref cache.StateRef) error {
	ret := _m.Called(ref)

	var r0 error
	if rf, ok := ret.Get(0).(func(cache.StateRef) error); ok {
		r0 = rf(ref)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetStateRef provides a mock function with given fields: network, cid, ifname
func (_m *StateCache) GetStateRef(network string, cid string, ifname string) cache.StateRef {
	ret := _m.Called(network, cid, ifname)

	var r0 cache.StateRef
	if rf, ok := ret.Get(0).(func(string, string, string) cache.StateRef); ok {
		r0 = rf(network, cid, ifname)
	} else {
		r0 = ret.Get(0).(cache.StateRef)
	}

	return r0
}

// Load provides a mock function with given fields: ref, state
func (_m *StateCache) Load(ref cache.StateRef, state interface{}) error {
	ret := _m.Called(ref, state)

	var r0 error
	if rf, ok := ret.Get(0).(func(cache.StateRef, interface{}) error); ok {
		r0 = rf(ref, state)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Save provides a mock function with given fields: ref, state
func (_m *StateCache) Save(ref cache.StateRef, state interface{}) error {
	ret := _m.Called(ref, state)

	var r0 error
	if rf, ok := ret.Get(0).(func(cache.StateRef, interface{}) error); ok {
		r0 = rf(ref, state)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
