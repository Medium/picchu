// Code generated by MockGen. DO NOT EDIT.
// Source: go.medium.engineering/picchu/controllers/releasemanager (interfaces: Incarnations)

// package controllers is a generated GoMock package.
package controllers

import (
	gomock "github.com/golang/mock/gomock"
	observe "go.medium.engineering/picchu/controllers/observe"
	reflect "reflect"
)

// MockIncarnations is a mock of Incarnations interface
type MockIncarnations struct {
	ctrl     *gomock.Controller
	recorder *MockIncarnationsMockRecorder
}

// MockIncarnationsMockRecorder is the mock recorder for MockIncarnations
type MockIncarnationsMockRecorder struct {
	mock *MockIncarnations
}

// NewMockIncarnations creates a new mock instance
func NewMockIncarnations(ctrl *gomock.Controller) *MockIncarnations {
	mock := &MockIncarnations{ctrl: ctrl}
	mock.recorder = &MockIncarnationsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockIncarnations) EXPECT() *MockIncarnationsMockRecorder {
	return m.recorder
}

// alertable mocks base method
func (m *MockIncarnations) alertable() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "alertable")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// alertable indicates an expected call of alertable
func (mr *MockIncarnationsMockRecorder) alertable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "alertable", reflect.TypeOf((*MockIncarnations)(nil).alertable))
}

// deployed mocks base method
func (m *MockIncarnations) deployed() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deployed")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// deployed indicates an expected call of deployed
func (mr *MockIncarnationsMockRecorder) deployed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deployed", reflect.TypeOf((*MockIncarnations)(nil).deployed))
}

// releasable mocks base method
func (m *MockIncarnations) releasable() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "releasable")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// releasable indicates an expected call of releasable
func (mr *MockIncarnationsMockRecorder) releasable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "releasable", reflect.TypeOf((*MockIncarnations)(nil).releasable))
}

// revisioned mocks base method
func (m *MockIncarnations) revisioned() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "revisioned")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// revisioned indicates an expected call of revisioned
func (mr *MockIncarnationsMockRecorder) revisioned() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "revisioned", reflect.TypeOf((*MockIncarnations)(nil).revisioned))
}

// sorted mocks base method
func (m *MockIncarnations) sorted() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "sorted")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// sorted indicates an expected call of sorted
func (mr *MockIncarnationsMockRecorder) sorted() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sorted", reflect.TypeOf((*MockIncarnations)(nil).sorted))
}

// unreleasable mocks base method
func (m *MockIncarnations) unreleasable() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "unreleasable")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// unreleasable indicates an expected call of unreleasable
func (mr *MockIncarnationsMockRecorder) unreleasable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "unreleasable", reflect.TypeOf((*MockIncarnations)(nil).unreleasable))
}

// unretirable mocks base method
func (m *MockIncarnations) unretirable() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "unretirable")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// unretirable indicates an expected call of unretirable
func (mr *MockIncarnationsMockRecorder) unretirable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "unretirable", reflect.TypeOf((*MockIncarnations)(nil).unretirable))
}

// update mocks base method
func (m *MockIncarnations) update(arg0 *observe.Observation) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "update", arg0)
}

// update indicates an expected call of update
func (mr *MockIncarnationsMockRecorder) update(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "update", reflect.TypeOf((*MockIncarnations)(nil).update), arg0)
}

// willRelease mocks base method
func (m *MockIncarnations) willRelease() []*Incarnation {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "willRelease")
	ret0, _ := ret[0].([]*Incarnation)
	return ret0
}

// willRelease indicates an expected call of willRelease
func (mr *MockIncarnationsMockRecorder) willRelease() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "willRelease", reflect.TypeOf((*MockIncarnations)(nil).willRelease))
}