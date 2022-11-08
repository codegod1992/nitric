// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/nitrictech/nitric/pkg/ifaces/cloudtasks (interfaces: CloudtasksClient)

// Package mock_cloudtasks is a generated GoMock package.
package mock_cloudtasks

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	gax "github.com/googleapis/gax-go/v2"
	tasks "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

// MockCloudtasksClient is a mock of CloudtasksClient interface.
type MockCloudtasksClient struct {
	ctrl     *gomock.Controller
	recorder *MockCloudtasksClientMockRecorder
}

// MockCloudtasksClientMockRecorder is the mock recorder for MockCloudtasksClient.
type MockCloudtasksClientMockRecorder struct {
	mock *MockCloudtasksClient
}

// NewMockCloudtasksClient creates a new mock instance.
func NewMockCloudtasksClient(ctrl *gomock.Controller) *MockCloudtasksClient {
	mock := &MockCloudtasksClient{ctrl: ctrl}
	mock.recorder = &MockCloudtasksClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCloudtasksClient) EXPECT() *MockCloudtasksClientMockRecorder {
	return m.recorder
}

// CreateTask mocks base method.
func (m *MockCloudtasksClient) CreateTask(arg0 context.Context, arg1 *tasks.CreateTaskRequest, arg2 ...gax.CallOption) (*tasks.Task, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CreateTask", varargs...)
	ret0, _ := ret[0].(*tasks.Task)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateTask indicates an expected call of CreateTask.
func (mr *MockCloudtasksClientMockRecorder) CreateTask(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTask", reflect.TypeOf((*MockCloudtasksClient)(nil).CreateTask), varargs...)
}