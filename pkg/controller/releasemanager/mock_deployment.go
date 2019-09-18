// Code generated by MockGen. DO NOT EDIT.
// Source: state.go

// Package releasemanager is a generated GoMock package.
package releasemanager

import (
	context "context"
	logr "github.com/go-logr/logr"
	gomock "github.com/golang/mock/gomock"
	v1alpha1 "go.medium.engineering/picchu/pkg/apis/picchu/v1alpha1"
	reflect "reflect"
)

// MockDeployment is a mock of Deployment interface
type MockDeployment struct {
	ctrl     *gomock.Controller
	recorder *MockDeploymentMockRecorder
}

// MockDeploymentMockRecorder is the mock recorder for MockDeployment
type MockDeploymentMockRecorder struct {
	mock *MockDeployment
}

// NewMockDeployment creates a new mock instance
func NewMockDeployment(ctrl *gomock.Controller) *MockDeployment {
	mock := &MockDeployment{ctrl: ctrl}
	mock.recorder = &MockDeploymentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDeployment) EXPECT() *MockDeploymentMockRecorder {
	return m.recorder
}

// sync mocks base method
func (m *MockDeployment) sync(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "sync", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// sync indicates an expected call of sync
func (mr *MockDeploymentMockRecorder) sync(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sync", reflect.TypeOf((*MockDeployment)(nil).sync), arg0)
}

// retire mocks base method
func (m *MockDeployment) retire(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "retire", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// retire indicates an expected call of retire
func (mr *MockDeploymentMockRecorder) retire(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "retire", reflect.TypeOf((*MockDeployment)(nil).retire), arg0)
}

// del mocks base method
func (m *MockDeployment) del(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "del", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// del indicates an expected call of del
func (mr *MockDeploymentMockRecorder) del(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "del", reflect.TypeOf((*MockDeployment)(nil).del), arg0)
}

// syncCanaryRules mocks base method
func (m *MockDeployment) syncCanaryRules(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncCanaryRules", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncCanaryRules indicates an expected call of syncCanaryRules
func (mr *MockDeploymentMockRecorder) syncCanaryRules(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncCanaryRules", reflect.TypeOf((*MockDeployment)(nil).syncCanaryRules), arg0)
}

// deleteCanaryRules mocks base method
func (m *MockDeployment) deleteCanaryRules(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteCanaryRules", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteCanaryRules indicates an expected call of deleteCanaryRules
func (mr *MockDeploymentMockRecorder) deleteCanaryRules(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteCanaryRules", reflect.TypeOf((*MockDeployment)(nil).deleteCanaryRules), arg0)
}

// syncSLIRules mocks base method
func (m *MockDeployment) syncSLIRules(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncSLIRules", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncSLIRules indicates an expected call of syncSLIRules
func (mr *MockDeploymentMockRecorder) syncSLIRules(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncSLIRules", reflect.TypeOf((*MockDeployment)(nil).syncSLIRules), arg0)
}

// deleteSLIRules mocks base method
func (m *MockDeployment) deleteSLIRules(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteSLIRules", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteSLIRules indicates an expected call of deleteSLIRules
func (mr *MockDeploymentMockRecorder) deleteSLIRules(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteSLIRules", reflect.TypeOf((*MockDeployment)(nil).deleteSLIRules), arg0)
}

// hasRevision mocks base method
func (m *MockDeployment) hasRevision() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "hasRevision")
	ret0, _ := ret[0].(bool)
	return ret0
}

// hasRevision indicates an expected call of hasRevision
func (mr *MockDeploymentMockRecorder) hasRevision() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "hasRevision", reflect.TypeOf((*MockDeployment)(nil).hasRevision))
}

// schedulePermitsRelease mocks base method
func (m *MockDeployment) schedulePermitsRelease() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "schedulePermitsRelease")
	ret0, _ := ret[0].(bool)
	return ret0
}

// schedulePermitsRelease indicates an expected call of schedulePermitsRelease
func (mr *MockDeploymentMockRecorder) schedulePermitsRelease() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "schedulePermitsRelease", reflect.TypeOf((*MockDeployment)(nil).schedulePermitsRelease))
}

// isAlarmTriggered mocks base method
func (m *MockDeployment) isAlarmTriggered() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isAlarmTriggered")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isAlarmTriggered indicates an expected call of isAlarmTriggered
func (mr *MockDeploymentMockRecorder) isAlarmTriggered() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isAlarmTriggered", reflect.TypeOf((*MockDeployment)(nil).isAlarmTriggered))
}

// isReleaseEligible mocks base method
func (m *MockDeployment) isReleaseEligible() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isReleaseEligible")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isReleaseEligible indicates an expected call of isReleaseEligible
func (mr *MockDeploymentMockRecorder) isReleaseEligible() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isReleaseEligible", reflect.TypeOf((*MockDeployment)(nil).isReleaseEligible))
}

// getStatus mocks base method
func (m *MockDeployment) getStatus() *v1alpha1.ReleaseManagerRevisionStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getStatus")
	ret0, _ := ret[0].(*v1alpha1.ReleaseManagerRevisionStatus)
	return ret0
}

// getStatus indicates an expected call of getStatus
func (mr *MockDeploymentMockRecorder) getStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getStatus", reflect.TypeOf((*MockDeployment)(nil).getStatus))
}

// setState mocks base method
func (m *MockDeployment) setState(target string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "setState", target)
}

// setState indicates an expected call of setState
func (mr *MockDeploymentMockRecorder) setState(target interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "setState", reflect.TypeOf((*MockDeployment)(nil).setState), target)
}

// getLog mocks base method
func (m *MockDeployment) getLog() logr.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getLog")
	ret0, _ := ret[0].(logr.Logger)
	return ret0
}

// getLog indicates an expected call of getLog
func (mr *MockDeploymentMockRecorder) getLog() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getLog", reflect.TypeOf((*MockDeployment)(nil).getLog))
}

// isDeployed mocks base method
func (m *MockDeployment) isDeployed() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isDeployed")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isDeployed indicates an expected call of isDeployed
func (mr *MockDeploymentMockRecorder) isDeployed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isDeployed", reflect.TypeOf((*MockDeployment)(nil).isDeployed))
}

// isTestPending mocks base method
func (m *MockDeployment) isTestPending() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isTestPending")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isTestPending indicates an expected call of isTestPending
func (mr *MockDeploymentMockRecorder) isTestPending() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isTestPending", reflect.TypeOf((*MockDeployment)(nil).isTestPending))
}

// isTestStarted mocks base method
func (m *MockDeployment) isTestStarted() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isTestStarted")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isTestStarted indicates an expected call of isTestStarted
func (mr *MockDeploymentMockRecorder) isTestStarted() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isTestStarted", reflect.TypeOf((*MockDeployment)(nil).isTestStarted))
}

// currentPercent mocks base method
func (m *MockDeployment) currentPercent() uint32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "currentPercent")
	ret0, _ := ret[0].(uint32)
	return ret0
}

// currentPercent indicates an expected call of currentPercent
func (mr *MockDeploymentMockRecorder) currentPercent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "currentPercent", reflect.TypeOf((*MockDeployment)(nil).currentPercent))
}

// peakPercent mocks base method
func (m *MockDeployment) peakPercent() uint32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "peakPercent")
	ret0, _ := ret[0].(uint32)
	return ret0
}

// peakPercent indicates an expected call of peakPercent
func (mr *MockDeploymentMockRecorder) peakPercent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "peakPercent", reflect.TypeOf((*MockDeployment)(nil).peakPercent))
}

// isCanaryPending mocks base method
func (m *MockDeployment) isCanaryPending() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isCanaryPending")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isCanaryPending indicates an expected call of isCanaryPending
func (mr *MockDeploymentMockRecorder) isCanaryPending() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isCanaryPending", reflect.TypeOf((*MockDeployment)(nil).isCanaryPending))
}
