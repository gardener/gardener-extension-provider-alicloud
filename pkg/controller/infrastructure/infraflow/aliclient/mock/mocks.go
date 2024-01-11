// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient (interfaces: Actor,Factory)
//
// Generated by this command:
//
//	mockgen -package aliclient -destination=mocks.go github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient Actor,Factory
//

// Package aliclient is a generated GoMock package.
package aliclient

import (
	context "context"
	reflect "reflect"

	aliclient "github.com/gardener/gardener-extension-provider-alicloud/pkg/controller/infrastructure/infraflow/aliclient"
	gomock "go.uber.org/mock/gomock"
)

// MockActor is a mock of Actor interface.
type MockActor struct {
	ctrl     *gomock.Controller
	recorder *MockActorMockRecorder
}

// MockActorMockRecorder is the mock recorder for MockActor.
type MockActorMockRecorder struct {
	mock *MockActor
}

// NewMockActor creates a new mock instance.
func NewMockActor(ctrl *gomock.Controller) *MockActor {
	mock := &MockActor{ctrl: ctrl}
	mock.recorder = &MockActorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockActor) EXPECT() *MockActorMockRecorder {
	return m.recorder
}

// AssociateEIP mocks base method.
func (m *MockActor) AssociateEIP(arg0 context.Context, arg1, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AssociateEIP", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// AssociateEIP indicates an expected call of AssociateEIP.
func (mr *MockActorMockRecorder) AssociateEIP(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AssociateEIP", reflect.TypeOf((*MockActor)(nil).AssociateEIP), arg0, arg1, arg2, arg3)
}

// AuthorizeSecurityGroupRule mocks base method.
func (m *MockActor) AuthorizeSecurityGroupRule(arg0 context.Context, arg1 string, arg2 aliclient.SecurityGroupRule) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthorizeSecurityGroupRule", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// AuthorizeSecurityGroupRule indicates an expected call of AuthorizeSecurityGroupRule.
func (mr *MockActorMockRecorder) AuthorizeSecurityGroupRule(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthorizeSecurityGroupRule", reflect.TypeOf((*MockActor)(nil).AuthorizeSecurityGroupRule), arg0, arg1, arg2)
}

// CreateEIP mocks base method.
func (m *MockActor) CreateEIP(arg0 context.Context, arg1 *aliclient.EIP) (*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEIP", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEIP indicates an expected call of CreateEIP.
func (mr *MockActorMockRecorder) CreateEIP(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEIP", reflect.TypeOf((*MockActor)(nil).CreateEIP), arg0, arg1)
}

// CreateIpv6Gateway mocks base method.
func (m *MockActor) CreateIpv6Gateway(arg0 context.Context, arg1 *aliclient.IPV6Gateway) (*aliclient.IPV6Gateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateIpv6Gateway", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.IPV6Gateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateIpv6Gateway indicates an expected call of CreateIpv6Gateway.
func (mr *MockActorMockRecorder) CreateIpv6Gateway(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIpv6Gateway", reflect.TypeOf((*MockActor)(nil).CreateIpv6Gateway), arg0, arg1)
}

// CreateNatGateway mocks base method.
func (m *MockActor) CreateNatGateway(arg0 context.Context, arg1 *aliclient.NatGateway) (*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNatGateway", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNatGateway indicates an expected call of CreateNatGateway.
func (mr *MockActorMockRecorder) CreateNatGateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNatGateway", reflect.TypeOf((*MockActor)(nil).CreateNatGateway), arg0, arg1)
}

// CreateSNatEntry mocks base method.
func (m *MockActor) CreateSNatEntry(arg0 context.Context, arg1 *aliclient.SNATEntry) (*aliclient.SNATEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSNatEntry", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.SNATEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSNatEntry indicates an expected call of CreateSNatEntry.
func (mr *MockActorMockRecorder) CreateSNatEntry(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSNatEntry", reflect.TypeOf((*MockActor)(nil).CreateSNatEntry), arg0, arg1)
}

// CreateSecurityGroup mocks base method.
func (m *MockActor) CreateSecurityGroup(arg0 context.Context, arg1 *aliclient.SecurityGroup) (*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSecurityGroup", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSecurityGroup indicates an expected call of CreateSecurityGroup.
func (mr *MockActorMockRecorder) CreateSecurityGroup(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSecurityGroup", reflect.TypeOf((*MockActor)(nil).CreateSecurityGroup), arg0, arg1)
}

// CreateTags mocks base method.
func (m *MockActor) CreateTags(arg0 context.Context, arg1 []string, arg2 aliclient.Tags, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTags", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTags indicates an expected call of CreateTags.
func (mr *MockActorMockRecorder) CreateTags(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTags", reflect.TypeOf((*MockActor)(nil).CreateTags), arg0, arg1, arg2, arg3)
}

// CreateVSwitch mocks base method.
func (m *MockActor) CreateVSwitch(arg0 context.Context, arg1 *aliclient.VSwitch) (*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVSwitch", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVSwitch indicates an expected call of CreateVSwitch.
func (mr *MockActorMockRecorder) CreateVSwitch(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVSwitch", reflect.TypeOf((*MockActor)(nil).CreateVSwitch), arg0, arg1)
}

// CreateVpc mocks base method.
func (m *MockActor) CreateVpc(arg0 context.Context, arg1 *aliclient.VPC) (*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVpc", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVpc indicates an expected call of CreateVpc.
func (mr *MockActorMockRecorder) CreateVpc(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVpc", reflect.TypeOf((*MockActor)(nil).CreateVpc), arg0, arg1)
}

// DeleteEIP mocks base method.
func (m *MockActor) DeleteEIP(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEIP", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEIP indicates an expected call of DeleteEIP.
func (mr *MockActorMockRecorder) DeleteEIP(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEIP", reflect.TypeOf((*MockActor)(nil).DeleteEIP), arg0, arg1)
}

// DeleteIpv6Gateway mocks base method.
func (m *MockActor) DeleteIpv6Gateway(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteIpv6Gateway", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteIpv6Gateway indicates an expected call of DeleteIpv6Gateway.
func (mr *MockActorMockRecorder) DeleteIpv6Gateway(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteIpv6Gateway", reflect.TypeOf((*MockActor)(nil).DeleteIpv6Gateway), arg0, arg1)
}

// DeleteNatGateway mocks base method.
func (m *MockActor) DeleteNatGateway(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNatGateway", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNatGateway indicates an expected call of DeleteNatGateway.
func (mr *MockActorMockRecorder) DeleteNatGateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNatGateway", reflect.TypeOf((*MockActor)(nil).DeleteNatGateway), arg0, arg1)
}

// DeleteSNatEntry mocks base method.
func (m *MockActor) DeleteSNatEntry(arg0 context.Context, arg1, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSNatEntry", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSNatEntry indicates an expected call of DeleteSNatEntry.
func (mr *MockActorMockRecorder) DeleteSNatEntry(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSNatEntry", reflect.TypeOf((*MockActor)(nil).DeleteSNatEntry), arg0, arg1, arg2)
}

// DeleteSecurityGroup mocks base method.
func (m *MockActor) DeleteSecurityGroup(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecurityGroup", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecurityGroup indicates an expected call of DeleteSecurityGroup.
func (mr *MockActorMockRecorder) DeleteSecurityGroup(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecurityGroup", reflect.TypeOf((*MockActor)(nil).DeleteSecurityGroup), arg0, arg1)
}

// DeleteTags mocks base method.
func (m *MockActor) DeleteTags(arg0 context.Context, arg1 []string, arg2 aliclient.Tags, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTags", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTags indicates an expected call of DeleteTags.
func (mr *MockActorMockRecorder) DeleteTags(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTags", reflect.TypeOf((*MockActor)(nil).DeleteTags), arg0, arg1, arg2, arg3)
}

// DeleteVSwitch mocks base method.
func (m *MockActor) DeleteVSwitch(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVSwitch", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVSwitch indicates an expected call of DeleteVSwitch.
func (mr *MockActorMockRecorder) DeleteVSwitch(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVSwitch", reflect.TypeOf((*MockActor)(nil).DeleteVSwitch), arg0, arg1)
}

// DeleteVpc mocks base method.
func (m *MockActor) DeleteVpc(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVpc", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVpc indicates an expected call of DeleteVpc.
func (mr *MockActorMockRecorder) DeleteVpc(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVpc", reflect.TypeOf((*MockActor)(nil).DeleteVpc), arg0, arg1)
}

// FindEIPsByTags mocks base method.
func (m *MockActor) FindEIPsByTags(arg0 context.Context, arg1 aliclient.Tags) ([]*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindEIPsByTags", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindEIPsByTags indicates an expected call of FindEIPsByTags.
func (mr *MockActorMockRecorder) FindEIPsByTags(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindEIPsByTags", reflect.TypeOf((*MockActor)(nil).FindEIPsByTags), arg0, arg1)
}

// FindIpv6GatewaysByTags mocks base method.
func (m *MockActor) FindIpv6GatewaysByTags(arg0 context.Context, arg1 aliclient.Tags) ([]*aliclient.IPV6Gateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindIpv6GatewaysByTags", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.IPV6Gateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindIpv6GatewaysByTags indicates an expected call of FindIpv6GatewaysByTags.
func (mr *MockActorMockRecorder) FindIpv6GatewaysByTags(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindIpv6GatewaysByTags", reflect.TypeOf((*MockActor)(nil).FindIpv6GatewaysByTags), arg0, arg1)
}

// FindNatGatewayByTags mocks base method.
func (m *MockActor) FindNatGatewayByTags(arg0 context.Context, arg1 aliclient.Tags) ([]*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindNatGatewayByTags", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindNatGatewayByTags indicates an expected call of FindNatGatewayByTags.
func (mr *MockActorMockRecorder) FindNatGatewayByTags(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindNatGatewayByTags", reflect.TypeOf((*MockActor)(nil).FindNatGatewayByTags), arg0, arg1)
}

// FindNatGatewayByVPC mocks base method.
func (m *MockActor) FindNatGatewayByVPC(arg0 context.Context, arg1 string) (*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindNatGatewayByVPC", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindNatGatewayByVPC indicates an expected call of FindNatGatewayByVPC.
func (mr *MockActorMockRecorder) FindNatGatewayByVPC(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindNatGatewayByVPC", reflect.TypeOf((*MockActor)(nil).FindNatGatewayByVPC), arg0, arg1)
}

// FindSNatEntriesByNatGateway mocks base method.
func (m *MockActor) FindSNatEntriesByNatGateway(arg0 context.Context, arg1 string) ([]*aliclient.SNATEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindSNatEntriesByNatGateway", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.SNATEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindSNatEntriesByNatGateway indicates an expected call of FindSNatEntriesByNatGateway.
func (mr *MockActorMockRecorder) FindSNatEntriesByNatGateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindSNatEntriesByNatGateway", reflect.TypeOf((*MockActor)(nil).FindSNatEntriesByNatGateway), arg0, arg1)
}

// FindSecurityGroupsByTags mocks base method.
func (m *MockActor) FindSecurityGroupsByTags(arg0 context.Context, arg1 aliclient.Tags) ([]*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindSecurityGroupsByTags", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindSecurityGroupsByTags indicates an expected call of FindSecurityGroupsByTags.
func (mr *MockActorMockRecorder) FindSecurityGroupsByTags(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindSecurityGroupsByTags", reflect.TypeOf((*MockActor)(nil).FindSecurityGroupsByTags), arg0, arg1)
}

// FindVSwitchesByTags mocks base method.
func (m *MockActor) FindVSwitchesByTags(arg0 context.Context, arg1 aliclient.Tags) ([]*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindVSwitchesByTags", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindVSwitchesByTags indicates an expected call of FindVSwitchesByTags.
func (mr *MockActorMockRecorder) FindVSwitchesByTags(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindVSwitchesByTags", reflect.TypeOf((*MockActor)(nil).FindVSwitchesByTags), arg0, arg1)
}

// FindVpcsByTags mocks base method.
func (m *MockActor) FindVpcsByTags(arg0 context.Context, arg1 aliclient.Tags) ([]*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindVpcsByTags", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindVpcsByTags indicates an expected call of FindVpcsByTags.
func (mr *MockActorMockRecorder) FindVpcsByTags(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindVpcsByTags", reflect.TypeOf((*MockActor)(nil).FindVpcsByTags), arg0, arg1)
}

// GetEIP mocks base method.
func (m *MockActor) GetEIP(arg0 context.Context, arg1 string) (*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEIP", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEIP indicates an expected call of GetEIP.
func (mr *MockActorMockRecorder) GetEIP(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEIP", reflect.TypeOf((*MockActor)(nil).GetEIP), arg0, arg1)
}

// GetEIPByAddress mocks base method.
func (m *MockActor) GetEIPByAddress(arg0 context.Context, arg1 string) (*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEIPByAddress", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.EIP)
// GetIpv6Gateway mocks base method.
func (m *MockActor) GetIpv6Gateway(arg0 context.Context, arg1 string) (*aliclient.IPV6Gateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIpv6Gateway", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.IPV6Gateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEIPByAddress indicates an expected call of GetEIPByAddress.
func (mr *MockActorMockRecorder) GetEIPByAddress(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEIPByAddress", reflect.TypeOf((*MockActor)(nil).GetEIPByAddress), arg0, arg1)
// GetIpv6Gateway indicates an expected call of GetIpv6Gateway.
func (mr *MockActorMockRecorder) GetIpv6Gateway(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIpv6Gateway", reflect.TypeOf((*MockActor)(nil).GetIpv6Gateway), arg0, arg1)
}

// GetNatGateway mocks base method.
func (m *MockActor) GetNatGateway(arg0 context.Context, arg1 string) (*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNatGateway", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNatGateway indicates an expected call of GetNatGateway.
func (mr *MockActorMockRecorder) GetNatGateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNatGateway", reflect.TypeOf((*MockActor)(nil).GetNatGateway), arg0, arg1)
}

// GetSNatEntry mocks base method.
func (m *MockActor) GetSNatEntry(arg0 context.Context, arg1, arg2 string) (*aliclient.SNATEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSNatEntry", arg0, arg1, arg2)
	ret0, _ := ret[0].(*aliclient.SNATEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSNatEntry indicates an expected call of GetSNatEntry.
func (mr *MockActorMockRecorder) GetSNatEntry(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSNatEntry", reflect.TypeOf((*MockActor)(nil).GetSNatEntry), arg0, arg1, arg2)
}

// GetSecurityGroup mocks base method.
func (m *MockActor) GetSecurityGroup(arg0 context.Context, arg1 string) (*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecurityGroup", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecurityGroup indicates an expected call of GetSecurityGroup.
func (mr *MockActorMockRecorder) GetSecurityGroup(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecurityGroup", reflect.TypeOf((*MockActor)(nil).GetSecurityGroup), arg0, arg1)
}

// GetVSwitch mocks base method.
func (m *MockActor) GetVSwitch(arg0 context.Context, arg1 string) (*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVSwitch", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVSwitch indicates an expected call of GetVSwitch.
func (mr *MockActorMockRecorder) GetVSwitch(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVSwitch", reflect.TypeOf((*MockActor)(nil).GetVSwitch), arg0, arg1)
}

// GetVpc mocks base method.
func (m *MockActor) GetVpc(arg0 context.Context, arg1 string) (*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVpc", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVpc indicates an expected call of GetVpc.
func (mr *MockActorMockRecorder) GetVpc(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVpc", reflect.TypeOf((*MockActor)(nil).GetVpc), arg0, arg1)
}

// LisIpv6Gateways mocks base method.
func (m *MockActor) LisIpv6Gateways(arg0 context.Context, arg1 []string) ([]*aliclient.IPV6Gateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LisIpv6Gateways", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.IPV6Gateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LisIpv6Gateways indicates an expected call of LisIpv6Gateways.
func (mr *MockActorMockRecorder) LisIpv6Gateways(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LisIpv6Gateways", reflect.TypeOf((*MockActor)(nil).LisIpv6Gateways), arg0, arg1)
}

// ListEIPs mocks base method.
func (m *MockActor) ListEIPs(arg0 context.Context, arg1 []string) ([]*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEIPs", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEIPs indicates an expected call of ListEIPs.
func (mr *MockActorMockRecorder) ListEIPs(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEIPs", reflect.TypeOf((*MockActor)(nil).ListEIPs), arg0, arg1)
}

// ListEnhanhcedNatGatewayAvailableZones mocks base method.
func (m *MockActor) ListEnhanhcedNatGatewayAvailableZones(arg0 context.Context, arg1 string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEnhanhcedNatGatewayAvailableZones", arg0, arg1)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEnhanhcedNatGatewayAvailableZones indicates an expected call of ListEnhanhcedNatGatewayAvailableZones.
func (mr *MockActorMockRecorder) ListEnhanhcedNatGatewayAvailableZones(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEnhanhcedNatGatewayAvailableZones", reflect.TypeOf((*MockActor)(nil).ListEnhanhcedNatGatewayAvailableZones), arg0, arg1)
}

// ListNatGateways mocks base method.
func (m *MockActor) ListNatGateways(arg0 context.Context, arg1 []string) ([]*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNatGateways", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNatGateways indicates an expected call of ListNatGateways.
func (mr *MockActorMockRecorder) ListNatGateways(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNatGateways", reflect.TypeOf((*MockActor)(nil).ListNatGateways), arg0, arg1)
}

// ListSecurityGroups mocks base method.
func (m *MockActor) ListSecurityGroups(arg0 context.Context, arg1 []string) ([]*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListSecurityGroups", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSecurityGroups indicates an expected call of ListSecurityGroups.
func (mr *MockActorMockRecorder) ListSecurityGroups(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSecurityGroups", reflect.TypeOf((*MockActor)(nil).ListSecurityGroups), arg0, arg1)
}

// ListVSwitches mocks base method.
func (m *MockActor) ListVSwitches(arg0 context.Context, arg1 []string) ([]*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVSwitches", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVSwitches indicates an expected call of ListVSwitches.
func (mr *MockActorMockRecorder) ListVSwitches(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVSwitches", reflect.TypeOf((*MockActor)(nil).ListVSwitches), arg0, arg1)
}

// ListVpcs mocks base method.
func (m *MockActor) ListVpcs(arg0 context.Context, arg1 []string) ([]*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVpcs", arg0, arg1)
	ret0, _ := ret[0].([]*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVpcs indicates an expected call of ListVpcs.
func (mr *MockActorMockRecorder) ListVpcs(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVpcs", reflect.TypeOf((*MockActor)(nil).ListVpcs), arg0, arg1)
}

// ModifyEIP mocks base method.
func (m *MockActor) ModifyEIP(arg0 context.Context, arg1 string, arg2 *aliclient.EIP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModifyEIP", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ModifyEIP indicates an expected call of ModifyEIP.
func (mr *MockActorMockRecorder) ModifyEIP(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModifyEIP", reflect.TypeOf((*MockActor)(nil).ModifyEIP), arg0, arg1, arg2)
}

// ModifyVpc mocks base method.
func (m *MockActor) ModifyVpc(arg0 context.Context, arg1 string, arg2 *aliclient.VPC) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModifyVpc", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ModifyVpc indicates an expected call of ModifyVpc.
func (mr *MockActorMockRecorder) ModifyVpc(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModifyVpc", reflect.TypeOf((*MockActor)(nil).ModifyVpc), arg0, arg1, arg2)
}

// RevokeSecurityGroupRule mocks base method.
func (m *MockActor) RevokeSecurityGroupRule(arg0 context.Context, arg1, arg2, arg3 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevokeSecurityGroupRule", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// RevokeSecurityGroupRule indicates an expected call of RevokeSecurityGroupRule.
func (mr *MockActorMockRecorder) RevokeSecurityGroupRule(arg0, arg1, arg2, arg3 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevokeSecurityGroupRule", reflect.TypeOf((*MockActor)(nil).RevokeSecurityGroupRule), arg0, arg1, arg2, arg3)
}

// UnAssociateEIP mocks base method.
func (m *MockActor) UnAssociateEIP(arg0 context.Context, arg1 *aliclient.EIP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnAssociateEIP", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnAssociateEIP indicates an expected call of UnAssociateEIP.
func (mr *MockActorMockRecorder) UnAssociateEIP(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnAssociateEIP", reflect.TypeOf((*MockActor)(nil).UnAssociateEIP), arg0, arg1)
}

// MockFactory is a mock of Factory interface.
type MockFactory struct {
	ctrl     *gomock.Controller
	recorder *MockFactoryMockRecorder
}

// MockFactoryMockRecorder is the mock recorder for MockFactory.
type MockFactoryMockRecorder struct {
	mock *MockFactory
}

// NewMockFactory creates a new mock instance.
func NewMockFactory(ctrl *gomock.Controller) *MockFactory {
	mock := &MockFactory{ctrl: ctrl}
	mock.recorder = &MockFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFactory) EXPECT() *MockFactoryMockRecorder {
	return m.recorder
}

// NewActor mocks base method.
func (m *MockFactory) NewActor(arg0, arg1, arg2 string) (aliclient.Actor, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewActor", arg0, arg1, arg2)
	ret0, _ := ret[0].(aliclient.Actor)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewActor indicates an expected call of NewActor.
func (mr *MockFactoryMockRecorder) NewActor(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewActor", reflect.TypeOf((*MockFactory)(nil).NewActor), arg0, arg1, arg2)
}
