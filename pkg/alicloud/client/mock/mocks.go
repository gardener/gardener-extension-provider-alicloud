// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client (interfaces: DNS,ClientFactory)
//
// Generated by this command:
//
//	mockgen -package client -destination=mocks.go github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client DNS,ClientFactory
//

// Package client is a generated GoMock package.
package client

import (
	context "context"
	reflect "reflect"

	client "github.com/gardener/gardener-extension-provider-alicloud/pkg/alicloud/client"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	client0 "sigs.k8s.io/controller-runtime/pkg/client"
)

// MockDNS is a mock of DNS interface.
type MockDNS struct {
	ctrl     *gomock.Controller
	recorder *MockDNSMockRecorder
}

// MockDNSMockRecorder is the mock recorder for MockDNS.
type MockDNSMockRecorder struct {
	mock *MockDNS
}

// NewMockDNS creates a new mock instance.
func NewMockDNS(ctrl *gomock.Controller) *MockDNS {
	mock := &MockDNS{ctrl: ctrl}
	mock.recorder = &MockDNSMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDNS) EXPECT() *MockDNSMockRecorder {
	return m.recorder
}

// CreateOrUpdateDomainRecords mocks base method.
func (m *MockDNS) CreateOrUpdateDomainRecords(arg0 context.Context, arg1, arg2, arg3 string, arg4 []string, arg5 int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrUpdateDomainRecords", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateOrUpdateDomainRecords indicates an expected call of CreateOrUpdateDomainRecords.
func (mr *MockDNSMockRecorder) CreateOrUpdateDomainRecords(arg0, arg1, arg2, arg3, arg4, arg5 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrUpdateDomainRecords", reflect.TypeOf((*MockDNS)(nil).CreateOrUpdateDomainRecords), arg0, arg1, arg2, arg3, arg4, arg5)
}

// DeleteDomainRecords mocks base method.
func (m *MockDNS) DeleteDomainRecords(arg0 context.Context, arg1, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteDomainRecords", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteDomainRecords indicates an expected call of DeleteDomainRecords.
func (mr *MockDNSMockRecorder) DeleteDomainRecords(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteDomainRecords", reflect.TypeOf((*MockDNS)(nil).DeleteDomainRecords), arg0, arg1, arg2, arg3)
}

// GetDomainName mocks base method.
func (m *MockDNS) GetDomainName(arg0 context.Context, arg1 string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomainName", arg0, arg1)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomainName indicates an expected call of GetDomainName.
func (mr *MockDNSMockRecorder) GetDomainName(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomainName", reflect.TypeOf((*MockDNS)(nil).GetDomainName), arg0, arg1)
}

// GetDomainNames mocks base method.
func (m *MockDNS) GetDomainNames(arg0 context.Context) (map[string]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomainNames", arg0)
	ret0, _ := ret[0].(map[string]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomainNames indicates an expected call of GetDomainNames.
func (mr *MockDNSMockRecorder) GetDomainNames(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomainNames", reflect.TypeOf((*MockDNS)(nil).GetDomainNames), arg0)
}

// MockClientFactory is a mock of ClientFactory interface.
type MockClientFactory struct {
	ctrl     *gomock.Controller
	recorder *MockClientFactoryMockRecorder
}

// MockClientFactoryMockRecorder is the mock recorder for MockClientFactory.
type MockClientFactoryMockRecorder struct {
	mock *MockClientFactory
}

// NewMockClientFactory creates a new mock instance.
func NewMockClientFactory(ctrl *gomock.Controller) *MockClientFactory {
	mock := &MockClientFactory{ctrl: ctrl}
	mock.recorder = &MockClientFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientFactory) EXPECT() *MockClientFactoryMockRecorder {
	return m.recorder
}

// NewDNSClient mocks base method.
func (m *MockClientFactory) NewDNSClient(arg0, arg1, arg2 string) (client.DNS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewDNSClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.DNS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewDNSClient indicates an expected call of NewDNSClient.
func (mr *MockClientFactoryMockRecorder) NewDNSClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewDNSClient", reflect.TypeOf((*MockClientFactory)(nil).NewDNSClient), arg0, arg1, arg2)
}

// NewECSClient mocks base method.
func (m *MockClientFactory) NewECSClient(arg0, arg1, arg2 string) (client.ECS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewECSClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.ECS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewECSClient indicates an expected call of NewECSClient.
func (mr *MockClientFactoryMockRecorder) NewECSClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewECSClient", reflect.TypeOf((*MockClientFactory)(nil).NewECSClient), arg0, arg1, arg2)
}

// NewOSSClient mocks base method.
func (m *MockClientFactory) NewOSSClient(arg0, arg1, arg2 string) (client.OSS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewOSSClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.OSS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewOSSClient indicates an expected call of NewOSSClient.
func (mr *MockClientFactoryMockRecorder) NewOSSClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewOSSClient", reflect.TypeOf((*MockClientFactory)(nil).NewOSSClient), arg0, arg1, arg2)
}

// NewOSSClientFromSecretRef mocks base method.
func (m *MockClientFactory) NewOSSClientFromSecretRef(arg0 context.Context, arg1 client0.Client, arg2 *v1.SecretReference, arg3 string) (client.OSS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewOSSClientFromSecretRef", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(client.OSS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewOSSClientFromSecretRef indicates an expected call of NewOSSClientFromSecretRef.
func (mr *MockClientFactoryMockRecorder) NewOSSClientFromSecretRef(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewOSSClientFromSecretRef", reflect.TypeOf((*MockClientFactory)(nil).NewOSSClientFromSecretRef), arg0, arg1, arg2, arg3)
}

// NewRAMClient mocks base method.
func (m *MockClientFactory) NewRAMClient(arg0, arg1, arg2 string) (client.RAM, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewRAMClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.RAM)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewRAMClient indicates an expected call of NewRAMClient.
func (mr *MockClientFactoryMockRecorder) NewRAMClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewRAMClient", reflect.TypeOf((*MockClientFactory)(nil).NewRAMClient), arg0, arg1, arg2)
}

// NewROSClient mocks base method.
func (m *MockClientFactory) NewROSClient(arg0, arg1, arg2 string) (client.ROS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewROSClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.ROS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewROSClient indicates an expected call of NewROSClient.
func (mr *MockClientFactoryMockRecorder) NewROSClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewROSClient", reflect.TypeOf((*MockClientFactory)(nil).NewROSClient), arg0, arg1, arg2)
}

// NewSLBClient mocks base method.
func (m *MockClientFactory) NewSLBClient(arg0, arg1, arg2 string) (client.SLB, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewSLBClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.SLB)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewSLBClient indicates an expected call of NewSLBClient.
func (mr *MockClientFactoryMockRecorder) NewSLBClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewSLBClient", reflect.TypeOf((*MockClientFactory)(nil).NewSLBClient), arg0, arg1, arg2)
}

// NewSTSClient mocks base method.
func (m *MockClientFactory) NewSTSClient(arg0, arg1, arg2 string) (client.STS, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewSTSClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.STS)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewSTSClient indicates an expected call of NewSTSClient.
func (mr *MockClientFactoryMockRecorder) NewSTSClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewSTSClient", reflect.TypeOf((*MockClientFactory)(nil).NewSTSClient), arg0, arg1, arg2)
}

// NewVPCClient mocks base method.
func (m *MockClientFactory) NewVPCClient(arg0, arg1, arg2 string) (client.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewVPCClient", arg0, arg1, arg2)
	ret0, _ := ret[0].(client.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewVPCClient indicates an expected call of NewVPCClient.
func (mr *MockClientFactoryMockRecorder) NewVPCClient(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewVPCClient", reflect.TypeOf((*MockClientFactory)(nil).NewVPCClient), arg0, arg1, arg2)
}
