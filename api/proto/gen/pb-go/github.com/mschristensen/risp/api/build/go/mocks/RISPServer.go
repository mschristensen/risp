// Code generated by mockery v2.12.1. DO NOT EDIT.

package mocks

import (
	_go "risp/api/proto/gen/pb-go/github.com/mschristensen/risp/api/build/go"

	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// RISPServer is an autogenerated mock type for the RISPServer type
type RISPServer struct {
	mock.Mock
}

// Connect provides a mock function with given fields: _a0
func (_m *RISPServer) Connect(_a0 _go.RISP_ConnectServer) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(_go.RISP_ConnectServer) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewRISPServer creates a new instance of RISPServer. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewRISPServer(t testing.TB) *RISPServer {
	mock := &RISPServer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
