// Code generated by MockGen. DO NOT EDIT.
// Source: go.medium.engineering/picchu/controllers/releasemanager/scaling (interfaces: ScalableTarget)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	v1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	reflect "reflect"
	time "time"
)

// MockScalableTarget is a mock of ScalableTarget interface
type MockScalableTarget struct {
	ctrl     *gomock.Controller
	recorder *MockScalableTargetMockRecorder
}

// MockScalableTargetMockRecorder is the mock recorder for MockScalableTarget
type MockScalableTargetMockRecorder struct {
	mock *MockScalableTarget
}

// NewMockScalableTarget creates a new mock instance
func NewMockScalableTarget(ctrl *gomock.Controller) *MockScalableTarget {
	mock := &MockScalableTarget{ctrl: ctrl}
	mock.recorder = &MockScalableTargetMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockScalableTarget) EXPECT() *MockScalableTargetMockRecorder {
	return m.recorder
}

// CanRampTo mocks base method
func (m *MockScalableTarget) CanRampTo(arg0 uint32) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CanRampTo", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// CanRampTo indicates an expected call of CanRampTo
func (mr *MockScalableTargetMockRecorder) CanRampTo(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CanRampTo", reflect.TypeOf((*MockScalableTarget)(nil).CanRampTo), arg0)
}

// CurrentPercent mocks base method
func (m *MockScalableTarget) CurrentPercent() uint32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CurrentPercent")
	ret0, _ := ret[0].(uint32)
	return ret0
}

// CurrentPercent indicates an expected call of CurrentPercent
func (mr *MockScalableTargetMockRecorder) CurrentPercent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CurrentPercent", reflect.TypeOf((*MockScalableTarget)(nil).CurrentPercent))
}

// LastUpdated mocks base method
func (m *MockScalableTarget) LastUpdated() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LastUpdated")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// LastUpdated indicates an expected call of LastUpdated
func (mr *MockScalableTargetMockRecorder) LastUpdated() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LastUpdated", reflect.TypeOf((*MockScalableTarget)(nil).LastUpdated))
}

// PeakPercent mocks base method
func (m *MockScalableTarget) PeakPercent() uint32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PeakPercent")
	ret0, _ := ret[0].(uint32)
	return ret0
}

// PeakPercent indicates an expected call of PeakPercent
func (mr *MockScalableTargetMockRecorder) PeakPercent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PeakPercent", reflect.TypeOf((*MockScalableTarget)(nil).PeakPercent))
}

// ReleaseInfo mocks base method
func (m *MockScalableTarget) ReleaseInfo() v1alpha1.ReleaseInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ReleaseInfo")
	ret0, _ := ret[0].(v1alpha1.ReleaseInfo)
	return ret0
}

// ReleaseInfo indicates an expected call of ReleaseInfo
func (mr *MockScalableTargetMockRecorder) ReleaseInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReleaseInfo", reflect.TypeOf((*MockScalableTarget)(nil).ReleaseInfo))
}
