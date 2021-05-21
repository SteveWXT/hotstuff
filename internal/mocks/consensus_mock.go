// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/relab/hotstuff/consensus (interfaces: Consensus)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	consensus "github.com/relab/hotstuff/consensus"
)

// MockConsensus is a mock of Consensus interface.
type MockConsensus struct {
	ctrl     *gomock.Controller
	recorder *MockConsensusMockRecorder
}

// MockConsensusMockRecorder is the mock recorder for MockConsensus.
type MockConsensusMockRecorder struct {
	mock *MockConsensus
}

// NewMockConsensus creates a new mock instance.
func NewMockConsensus(ctrl *gomock.Controller) *MockConsensus {
	mock := &MockConsensus{ctrl: ctrl}
	mock.recorder = &MockConsensusMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConsensus) EXPECT() *MockConsensusMockRecorder {
	return m.recorder
}

// OnPropose mocks base method.
func (m *MockConsensus) OnPropose(arg0 consensus.ProposeMsg) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnPropose", arg0)
}

// OnPropose indicates an expected call of OnPropose.
func (mr *MockConsensusMockRecorder) OnPropose(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnPropose", reflect.TypeOf((*MockConsensus)(nil).OnPropose), arg0)
}

// Propose mocks base method.
func (m *MockConsensus) Propose(arg0 consensus.SyncInfo) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Propose", arg0)
}

// Propose indicates an expected call of Propose.
func (mr *MockConsensusMockRecorder) Propose(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Propose", reflect.TypeOf((*MockConsensus)(nil).Propose), arg0)
}

// StopVoting mocks base method.
func (m *MockConsensus) StopVoting(arg0 consensus.View) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "StopVoting", arg0)
}

// StopVoting indicates an expected call of StopVoting.
func (mr *MockConsensusMockRecorder) StopVoting(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopVoting", reflect.TypeOf((*MockConsensus)(nil).StopVoting), arg0)
}
