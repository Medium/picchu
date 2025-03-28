// Code generated by MockGen. DO NOT EDIT.
// Source: go.medium.engineering/picchu/controllers (interfaces: Deployment)
//
// Generated by this command:
//
//	mockgen --build_flags=--mod=mod -destination=controllers/mock_deployment.go -package=controllers go.medium.engineering/picchu/controllers Deployment
//

// Package controllers is a generated GoMock package.
package controllers

import (
	context "context"
	reflect "reflect"

	logr "github.com/go-logr/logr"
	v1alpha1 "go.medium.engineering/picchu/api/v1alpha1"
	gomock "go.uber.org/mock/gomock"
)

// MockDeployment is a mock of Deployment interface.
type MockDeployment struct {
	ctrl     *gomock.Controller
	recorder *MockDeploymentMockRecorder
}

// MockDeploymentMockRecorder is the mock recorder for MockDeployment.
type MockDeploymentMockRecorder struct {
	mock *MockDeployment
}

// NewMockDeployment creates a new mock instance.
func NewMockDeployment(ctrl *gomock.Controller) *MockDeployment {
	mock := &MockDeployment{ctrl: ctrl}
	mock.recorder = &MockDeploymentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDeployment) EXPECT() *MockDeploymentMockRecorder {
	return m.recorder
}

// currentPercent mocks base method.
func (m *MockDeployment) currentPercent() uint32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "currentPercent")
	ret0, _ := ret[0].(uint32)
	return ret0
}

// currentPercent indicates an expected call of currentPercent.
func (mr *MockDeploymentMockRecorder) currentPercent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "currentPercent", reflect.TypeOf((*MockDeployment)(nil).currentPercent))
}

// del mocks base method.
func (m *MockDeployment) del(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "del", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// del indicates an expected call of del.
func (mr *MockDeploymentMockRecorder) del(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "del", reflect.TypeOf((*MockDeployment)(nil).del), arg0)
}

// deleteCanaryRules mocks base method.
func (m *MockDeployment) deleteCanaryRules(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteCanaryRules", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteCanaryRules indicates an expected call of deleteCanaryRules.
func (mr *MockDeploymentMockRecorder) deleteCanaryRules(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteCanaryRules", reflect.TypeOf((*MockDeployment)(nil).deleteCanaryRules), arg0)
}

// deleteDatadogCanaryMonitors mocks base method.
func (m *MockDeployment) deleteDatadogCanaryMonitors(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteDatadogCanaryMonitors", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteDatadogCanaryMonitors indicates an expected call of deleteDatadogCanaryMonitors.
func (mr *MockDeploymentMockRecorder) deleteDatadogCanaryMonitors(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteDatadogCanaryMonitors", reflect.TypeOf((*MockDeployment)(nil).deleteDatadogCanaryMonitors), arg0)
}

// deleteDatadogSLOs mocks base method.
func (m *MockDeployment) deleteDatadogSLOs(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteDatadogSLOs", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteDatadogSLOs indicates an expected call of deleteDatadogSLOs.
func (mr *MockDeploymentMockRecorder) deleteDatadogSLOs(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteDatadogSLOs", reflect.TypeOf((*MockDeployment)(nil).deleteDatadogSLOs), arg0)
}

// deleteTaggedServiceLevels mocks base method.
func (m *MockDeployment) deleteTaggedServiceLevels(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "deleteTaggedServiceLevels", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// deleteTaggedServiceLevels indicates an expected call of deleteTaggedServiceLevels.
func (mr *MockDeploymentMockRecorder) deleteTaggedServiceLevels(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "deleteTaggedServiceLevels", reflect.TypeOf((*MockDeployment)(nil).deleteTaggedServiceLevels), arg0)
}

// getExternalTestStatus mocks base method.
func (m *MockDeployment) getExternalTestStatus() ExternalTestStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getExternalTestStatus")
	ret0, _ := ret[0].(ExternalTestStatus)
	return ret0
}

// getExternalTestStatus indicates an expected call of getExternalTestStatus.
func (mr *MockDeploymentMockRecorder) getExternalTestStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getExternalTestStatus", reflect.TypeOf((*MockDeployment)(nil).getExternalTestStatus))
}

// getLog mocks base method.
func (m *MockDeployment) getLog() logr.Logger {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getLog")
	ret0, _ := ret[0].(logr.Logger)
	return ret0
}

// getLog indicates an expected call of getLog.
func (mr *MockDeploymentMockRecorder) getLog() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getLog", reflect.TypeOf((*MockDeployment)(nil).getLog))
}

// getStatus mocks base method.
func (m *MockDeployment) getStatus() *v1alpha1.ReleaseManagerRevisionStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getStatus")
	ret0, _ := ret[0].(*v1alpha1.ReleaseManagerRevisionStatus)
	return ret0
}

// getStatus indicates an expected call of getStatus.
func (mr *MockDeploymentMockRecorder) getStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getStatus", reflect.TypeOf((*MockDeployment)(nil).getStatus))
}

// hasRevision mocks base method.
func (m *MockDeployment) hasRevision() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "hasRevision")
	ret0, _ := ret[0].(bool)
	return ret0
}

// hasRevision indicates an expected call of hasRevision.
func (mr *MockDeploymentMockRecorder) hasRevision() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "hasRevision", reflect.TypeOf((*MockDeployment)(nil).hasRevision))
}

// isCanaryPending mocks base method.
func (m *MockDeployment) isCanaryPending() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isCanaryPending")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isCanaryPending indicates an expected call of isCanaryPending.
func (mr *MockDeploymentMockRecorder) isCanaryPending() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isCanaryPending", reflect.TypeOf((*MockDeployment)(nil).isCanaryPending))
}

// isDeployed mocks base method.
func (m *MockDeployment) isDeployed() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isDeployed")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isDeployed indicates an expected call of isDeployed.
func (mr *MockDeploymentMockRecorder) isDeployed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isDeployed", reflect.TypeOf((*MockDeployment)(nil).isDeployed))
}

// isExpired mocks base method.
func (m *MockDeployment) isExpired() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isExpired")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isExpired indicates an expected call of isExpired.
func (mr *MockDeploymentMockRecorder) isExpired() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isExpired", reflect.TypeOf((*MockDeployment)(nil).isExpired))
}

// isReleaseEligible mocks base method.
func (m *MockDeployment) isReleaseEligible() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isReleaseEligible")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isReleaseEligible indicates an expected call of isReleaseEligible.
func (mr *MockDeploymentMockRecorder) isReleaseEligible() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isReleaseEligible", reflect.TypeOf((*MockDeployment)(nil).isReleaseEligible))
}

// isTimingOut mocks base method.
func (m *MockDeployment) isTimingOut() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "isTimingOut")
	ret0, _ := ret[0].(bool)
	return ret0
}

// isTimingOut indicates an expected call of isTimingOut.
func (mr *MockDeploymentMockRecorder) isTimingOut() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "isTimingOut", reflect.TypeOf((*MockDeployment)(nil).isTimingOut))
}

// markedAsFailed mocks base method.
func (m *MockDeployment) markedAsFailed() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "markedAsFailed")
	ret0, _ := ret[0].(bool)
	return ret0
}

// markedAsFailed indicates an expected call of markedAsFailed.
func (mr *MockDeploymentMockRecorder) markedAsFailed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "markedAsFailed", reflect.TypeOf((*MockDeployment)(nil).markedAsFailed))
}

// peakPercent mocks base method.
func (m *MockDeployment) peakPercent() uint32 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "peakPercent")
	ret0, _ := ret[0].(uint32)
	return ret0
}

// peakPercent indicates an expected call of peakPercent.
func (mr *MockDeploymentMockRecorder) peakPercent() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "peakPercent", reflect.TypeOf((*MockDeployment)(nil).peakPercent))
}

// retire mocks base method.
func (m *MockDeployment) retire(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "retire", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// retire indicates an expected call of retire.
func (mr *MockDeploymentMockRecorder) retire(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "retire", reflect.TypeOf((*MockDeployment)(nil).retire), arg0)
}

// schedulePermitsRelease mocks base method.
func (m *MockDeployment) schedulePermitsRelease() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "schedulePermitsRelease")
	ret0, _ := ret[0].(bool)
	return ret0
}

// schedulePermitsRelease indicates an expected call of schedulePermitsRelease.
func (mr *MockDeploymentMockRecorder) schedulePermitsRelease() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "schedulePermitsRelease", reflect.TypeOf((*MockDeployment)(nil).schedulePermitsRelease))
}

// setState mocks base method.
func (m *MockDeployment) setState(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "setState", arg0)
}

// setState indicates an expected call of setState.
func (mr *MockDeploymentMockRecorder) setState(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "setState", reflect.TypeOf((*MockDeployment)(nil).setState), arg0)
}

// sync mocks base method.
func (m *MockDeployment) sync(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "sync", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// sync indicates an expected call of sync.
func (mr *MockDeploymentMockRecorder) sync(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "sync", reflect.TypeOf((*MockDeployment)(nil).sync), arg0)
}

// syncCanaryRules mocks base method.
func (m *MockDeployment) syncCanaryRules(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncCanaryRules", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncCanaryRules indicates an expected call of syncCanaryRules.
func (mr *MockDeploymentMockRecorder) syncCanaryRules(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncCanaryRules", reflect.TypeOf((*MockDeployment)(nil).syncCanaryRules), arg0)
}

// syncDatadogCanaryMonitors mocks base method.
func (m *MockDeployment) syncDatadogCanaryMonitors(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncDatadogCanaryMonitors", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncDatadogCanaryMonitors indicates an expected call of syncDatadogCanaryMonitors.
func (mr *MockDeploymentMockRecorder) syncDatadogCanaryMonitors(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncDatadogCanaryMonitors", reflect.TypeOf((*MockDeployment)(nil).syncDatadogCanaryMonitors), arg0)
}

// syncDatadogSLOs mocks base method.
func (m *MockDeployment) syncDatadogSLOs(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncDatadogSLOs", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncDatadogSLOs indicates an expected call of syncDatadogSLOs.
func (mr *MockDeploymentMockRecorder) syncDatadogSLOs(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncDatadogSLOs", reflect.TypeOf((*MockDeployment)(nil).syncDatadogSLOs), arg0)
}

// syncTaggedServiceLevels mocks base method.
func (m *MockDeployment) syncTaggedServiceLevels(arg0 context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "syncTaggedServiceLevels", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// syncTaggedServiceLevels indicates an expected call of syncTaggedServiceLevels.
func (mr *MockDeploymentMockRecorder) syncTaggedServiceLevels(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "syncTaggedServiceLevels", reflect.TypeOf((*MockDeployment)(nil).syncTaggedServiceLevels), arg0)
}
