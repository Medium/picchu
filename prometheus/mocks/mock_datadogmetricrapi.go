// Code generated by MockGen. DO NOT EDIT.
// Source: go.medium.engineering/picchu/datadog (interfaces: DatadogMetricAPI)
//
// Generated by this command:
//
//	mockgen --build_flags=--mod=mod -destination=prometheus/mocks/mock_datadogmetricrapi.go -package=mocks go.medium.engineering/picchu/datadog DatadogMetricAPI
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	http "net/http"
	reflect "reflect"

	datadogV1 "github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	gomock "go.uber.org/mock/gomock"
)

// MockDatadogMetricAPI is a mock of DatadogMetricAPI interface.
type MockDatadogMetricAPI struct {
	ctrl     *gomock.Controller
	recorder *MockDatadogMetricAPIMockRecorder
}

// MockDatadogMetricAPIMockRecorder is the mock recorder for MockDatadogMetricAPI.
type MockDatadogMetricAPIMockRecorder struct {
	mock *MockDatadogMetricAPI
}

// NewMockDatadogMetricAPI creates a new mock instance.
func NewMockDatadogMetricAPI(ctrl *gomock.Controller) *MockDatadogMetricAPI {
	mock := &MockDatadogMetricAPI{ctrl: ctrl}
	mock.recorder = &MockDatadogMetricAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDatadogMetricAPI) EXPECT() *MockDatadogMetricAPIMockRecorder {
	return m.recorder
}

// QueryMetrics mocks base method.
func (m *MockDatadogMetricAPI) QueryMetrics(arg0 context.Context, arg1, arg2 int64, arg3 string) (datadogV1.MetricsQueryResponse, *http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryMetrics", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(datadogV1.MetricsQueryResponse)
	ret1, _ := ret[1].(*http.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// QueryMetrics indicates an expected call of QueryMetrics.
func (mr *MockDatadogMetricAPIMockRecorder) QueryMetrics(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryMetrics", reflect.TypeOf((*MockDatadogMetricAPI)(nil).QueryMetrics), arg0, arg1, arg2, arg3)
}
