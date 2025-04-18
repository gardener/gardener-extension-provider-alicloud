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
	isgomock struct{}
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
func (m *MockActor) AssociateEIP(ctx context.Context, id, to, insType string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AssociateEIP", ctx, id, to, insType)
	ret0, _ := ret[0].(error)
	return ret0
}

// AssociateEIP indicates an expected call of AssociateEIP.
func (mr *MockActorMockRecorder) AssociateEIP(ctx, id, to, insType any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AssociateEIP", reflect.TypeOf((*MockActor)(nil).AssociateEIP), ctx, id, to, insType)
}

// AuthorizeSecurityGroupRule mocks base method.
func (m *MockActor) AuthorizeSecurityGroupRule(ctx context.Context, sgId string, rule aliclient.SecurityGroupRule) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AuthorizeSecurityGroupRule", ctx, sgId, rule)
	ret0, _ := ret[0].(error)
	return ret0
}

// AuthorizeSecurityGroupRule indicates an expected call of AuthorizeSecurityGroupRule.
func (mr *MockActorMockRecorder) AuthorizeSecurityGroupRule(ctx, sgId, rule any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AuthorizeSecurityGroupRule", reflect.TypeOf((*MockActor)(nil).AuthorizeSecurityGroupRule), ctx, sgId, rule)
}

// CreateEIP mocks base method.
func (m *MockActor) CreateEIP(ctx context.Context, eip *aliclient.EIP) (*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateEIP", ctx, eip)
	ret0, _ := ret[0].(*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateEIP indicates an expected call of CreateEIP.
func (mr *MockActorMockRecorder) CreateEIP(ctx, eip any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateEIP", reflect.TypeOf((*MockActor)(nil).CreateEIP), ctx, eip)
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
func (mr *MockActorMockRecorder) CreateIpv6Gateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateIpv6Gateway", reflect.TypeOf((*MockActor)(nil).CreateIpv6Gateway), arg0, arg1)
}

// CreateNatGateway mocks base method.
func (m *MockActor) CreateNatGateway(ctx context.Context, ngw *aliclient.NatGateway) (*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateNatGateway", ctx, ngw)
	ret0, _ := ret[0].(*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateNatGateway indicates an expected call of CreateNatGateway.
func (mr *MockActorMockRecorder) CreateNatGateway(ctx, ngw any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateNatGateway", reflect.TypeOf((*MockActor)(nil).CreateNatGateway), ctx, ngw)
}

// CreateSNatEntry mocks base method.
func (m *MockActor) CreateSNatEntry(ctx context.Context, entry *aliclient.SNATEntry) (*aliclient.SNATEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSNatEntry", ctx, entry)
	ret0, _ := ret[0].(*aliclient.SNATEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSNatEntry indicates an expected call of CreateSNatEntry.
func (mr *MockActorMockRecorder) CreateSNatEntry(ctx, entry any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSNatEntry", reflect.TypeOf((*MockActor)(nil).CreateSNatEntry), ctx, entry)
}

// CreateSecurityGroup mocks base method.
func (m *MockActor) CreateSecurityGroup(ctx context.Context, sg *aliclient.SecurityGroup) (*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateSecurityGroup", ctx, sg)
	ret0, _ := ret[0].(*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateSecurityGroup indicates an expected call of CreateSecurityGroup.
func (mr *MockActorMockRecorder) CreateSecurityGroup(ctx, sg any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateSecurityGroup", reflect.TypeOf((*MockActor)(nil).CreateSecurityGroup), ctx, sg)
}

// CreateTags mocks base method.
func (m *MockActor) CreateTags(ctx context.Context, resources []string, tags aliclient.Tags, resourceType string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTags", ctx, resources, tags, resourceType)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTags indicates an expected call of CreateTags.
func (mr *MockActorMockRecorder) CreateTags(ctx, resources, tags, resourceType any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTags", reflect.TypeOf((*MockActor)(nil).CreateTags), ctx, resources, tags, resourceType)
}

// CreateVSwitch mocks base method.
func (m *MockActor) CreateVSwitch(ctx context.Context, vsw *aliclient.VSwitch) (*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVSwitch", ctx, vsw)
	ret0, _ := ret[0].(*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVSwitch indicates an expected call of CreateVSwitch.
func (mr *MockActorMockRecorder) CreateVSwitch(ctx, vsw any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVSwitch", reflect.TypeOf((*MockActor)(nil).CreateVSwitch), ctx, vsw)
}

// CreateVpc mocks base method.
func (m *MockActor) CreateVpc(ctx context.Context, vpc *aliclient.VPC) (*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateVpc", ctx, vpc)
	ret0, _ := ret[0].(*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateVpc indicates an expected call of CreateVpc.
func (mr *MockActorMockRecorder) CreateVpc(ctx, vpc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateVpc", reflect.TypeOf((*MockActor)(nil).CreateVpc), ctx, vpc)
}

// DeleteEIP mocks base method.
func (m *MockActor) DeleteEIP(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteEIP", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteEIP indicates an expected call of DeleteEIP.
func (mr *MockActorMockRecorder) DeleteEIP(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteEIP", reflect.TypeOf((*MockActor)(nil).DeleteEIP), ctx, id)
}

// DeleteIpv6Gateway mocks base method.
func (m *MockActor) DeleteIpv6Gateway(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteIpv6Gateway", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteIpv6Gateway indicates an expected call of DeleteIpv6Gateway.
func (mr *MockActorMockRecorder) DeleteIpv6Gateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteIpv6Gateway", reflect.TypeOf((*MockActor)(nil).DeleteIpv6Gateway), arg0, arg1)
}

// DeleteNatGateway mocks base method.
func (m *MockActor) DeleteNatGateway(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteNatGateway", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteNatGateway indicates an expected call of DeleteNatGateway.
func (mr *MockActorMockRecorder) DeleteNatGateway(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteNatGateway", reflect.TypeOf((*MockActor)(nil).DeleteNatGateway), ctx, id)
}

// DeleteSNatEntry mocks base method.
func (m *MockActor) DeleteSNatEntry(ctx context.Context, id, snatTableId string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSNatEntry", ctx, id, snatTableId)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSNatEntry indicates an expected call of DeleteSNatEntry.
func (mr *MockActorMockRecorder) DeleteSNatEntry(ctx, id, snatTableId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSNatEntry", reflect.TypeOf((*MockActor)(nil).DeleteSNatEntry), ctx, id, snatTableId)
}

// DeleteSecurityGroup mocks base method.
func (m *MockActor) DeleteSecurityGroup(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteSecurityGroup", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteSecurityGroup indicates an expected call of DeleteSecurityGroup.
func (mr *MockActorMockRecorder) DeleteSecurityGroup(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteSecurityGroup", reflect.TypeOf((*MockActor)(nil).DeleteSecurityGroup), ctx, id)
}

// DeleteTags mocks base method.
func (m *MockActor) DeleteTags(ctx context.Context, resources []string, tags aliclient.Tags, resourceType string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTags", ctx, resources, tags, resourceType)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTags indicates an expected call of DeleteTags.
func (mr *MockActorMockRecorder) DeleteTags(ctx, resources, tags, resourceType any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTags", reflect.TypeOf((*MockActor)(nil).DeleteTags), ctx, resources, tags, resourceType)
}

// DeleteVSwitch mocks base method.
func (m *MockActor) DeleteVSwitch(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVSwitch", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVSwitch indicates an expected call of DeleteVSwitch.
func (mr *MockActorMockRecorder) DeleteVSwitch(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVSwitch", reflect.TypeOf((*MockActor)(nil).DeleteVSwitch), ctx, id)
}

// DeleteVpc mocks base method.
func (m *MockActor) DeleteVpc(ctx context.Context, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVpc", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVpc indicates an expected call of DeleteVpc.
func (mr *MockActorMockRecorder) DeleteVpc(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVpc", reflect.TypeOf((*MockActor)(nil).DeleteVpc), ctx, id)
}

// FindEIPsByTags mocks base method.
func (m *MockActor) FindEIPsByTags(ctx context.Context, tags aliclient.Tags) ([]*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindEIPsByTags", ctx, tags)
	ret0, _ := ret[0].([]*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindEIPsByTags indicates an expected call of FindEIPsByTags.
func (mr *MockActorMockRecorder) FindEIPsByTags(ctx, tags any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindEIPsByTags", reflect.TypeOf((*MockActor)(nil).FindEIPsByTags), ctx, tags)
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
func (mr *MockActorMockRecorder) FindIpv6GatewaysByTags(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindIpv6GatewaysByTags", reflect.TypeOf((*MockActor)(nil).FindIpv6GatewaysByTags), arg0, arg1)
}

// FindNatGatewayByTags mocks base method.
func (m *MockActor) FindNatGatewayByTags(ctx context.Context, tags aliclient.Tags) ([]*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindNatGatewayByTags", ctx, tags)
	ret0, _ := ret[0].([]*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindNatGatewayByTags indicates an expected call of FindNatGatewayByTags.
func (mr *MockActorMockRecorder) FindNatGatewayByTags(ctx, tags any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindNatGatewayByTags", reflect.TypeOf((*MockActor)(nil).FindNatGatewayByTags), ctx, tags)
}

// FindNatGatewayByVPC mocks base method.
func (m *MockActor) FindNatGatewayByVPC(ctx context.Context, vpcId string) (*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindNatGatewayByVPC", ctx, vpcId)
	ret0, _ := ret[0].(*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindNatGatewayByVPC indicates an expected call of FindNatGatewayByVPC.
func (mr *MockActorMockRecorder) FindNatGatewayByVPC(ctx, vpcId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindNatGatewayByVPC", reflect.TypeOf((*MockActor)(nil).FindNatGatewayByVPC), ctx, vpcId)
}

// FindSNatEntriesByNatGateway mocks base method.
func (m *MockActor) FindSNatEntriesByNatGateway(ctx context.Context, ngwId string) ([]*aliclient.SNATEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindSNatEntriesByNatGateway", ctx, ngwId)
	ret0, _ := ret[0].([]*aliclient.SNATEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindSNatEntriesByNatGateway indicates an expected call of FindSNatEntriesByNatGateway.
func (mr *MockActorMockRecorder) FindSNatEntriesByNatGateway(ctx, ngwId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindSNatEntriesByNatGateway", reflect.TypeOf((*MockActor)(nil).FindSNatEntriesByNatGateway), ctx, ngwId)
}

// FindSecurityGroupsByTags mocks base method.
func (m *MockActor) FindSecurityGroupsByTags(ctx context.Context, tags aliclient.Tags) ([]*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindSecurityGroupsByTags", ctx, tags)
	ret0, _ := ret[0].([]*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindSecurityGroupsByTags indicates an expected call of FindSecurityGroupsByTags.
func (mr *MockActorMockRecorder) FindSecurityGroupsByTags(ctx, tags any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindSecurityGroupsByTags", reflect.TypeOf((*MockActor)(nil).FindSecurityGroupsByTags), ctx, tags)
}

// FindVSwitchesByTags mocks base method.
func (m *MockActor) FindVSwitchesByTags(ctx context.Context, tags aliclient.Tags) ([]*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindVSwitchesByTags", ctx, tags)
	ret0, _ := ret[0].([]*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindVSwitchesByTags indicates an expected call of FindVSwitchesByTags.
func (mr *MockActorMockRecorder) FindVSwitchesByTags(ctx, tags any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindVSwitchesByTags", reflect.TypeOf((*MockActor)(nil).FindVSwitchesByTags), ctx, tags)
}

// FindVpcsByTags mocks base method.
func (m *MockActor) FindVpcsByTags(ctx context.Context, tags aliclient.Tags) ([]*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindVpcsByTags", ctx, tags)
	ret0, _ := ret[0].([]*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindVpcsByTags indicates an expected call of FindVpcsByTags.
func (mr *MockActorMockRecorder) FindVpcsByTags(ctx, tags any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindVpcsByTags", reflect.TypeOf((*MockActor)(nil).FindVpcsByTags), ctx, tags)
}

// GetEIP mocks base method.
func (m *MockActor) GetEIP(ctx context.Context, id string) (*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEIP", ctx, id)
	ret0, _ := ret[0].(*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEIP indicates an expected call of GetEIP.
func (mr *MockActorMockRecorder) GetEIP(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEIP", reflect.TypeOf((*MockActor)(nil).GetEIP), ctx, id)
}

// GetEIPByAddress mocks base method.
func (m *MockActor) GetEIPByAddress(ctx context.Context, ipAddress string) (*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEIPByAddress", ctx, ipAddress)
	ret0, _ := ret[0].(*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetEIPByAddress indicates an expected call of GetEIPByAddress.
func (mr *MockActorMockRecorder) GetEIPByAddress(ctx, ipAddress any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEIPByAddress", reflect.TypeOf((*MockActor)(nil).GetEIPByAddress), ctx, ipAddress)
}

// GetIpv6Gateway mocks base method.
func (m *MockActor) GetIpv6Gateway(arg0 context.Context, arg1 string) (*aliclient.IPV6Gateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIpv6Gateway", arg0, arg1)
	ret0, _ := ret[0].(*aliclient.IPV6Gateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetIpv6Gateway indicates an expected call of GetIpv6Gateway.
func (mr *MockActorMockRecorder) GetIpv6Gateway(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIpv6Gateway", reflect.TypeOf((*MockActor)(nil).GetIpv6Gateway), arg0, arg1)
}

// GetNatGateway mocks base method.
func (m *MockActor) GetNatGateway(ctx context.Context, id string) (*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNatGateway", ctx, id)
	ret0, _ := ret[0].(*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetNatGateway indicates an expected call of GetNatGateway.
func (mr *MockActorMockRecorder) GetNatGateway(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNatGateway", reflect.TypeOf((*MockActor)(nil).GetNatGateway), ctx, id)
}

// GetSNatEntry mocks base method.
func (m *MockActor) GetSNatEntry(ctx context.Context, id, snatTableId string) (*aliclient.SNATEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSNatEntry", ctx, id, snatTableId)
	ret0, _ := ret[0].(*aliclient.SNATEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSNatEntry indicates an expected call of GetSNatEntry.
func (mr *MockActorMockRecorder) GetSNatEntry(ctx, id, snatTableId any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSNatEntry", reflect.TypeOf((*MockActor)(nil).GetSNatEntry), ctx, id, snatTableId)
}

// GetSecurityGroup mocks base method.
func (m *MockActor) GetSecurityGroup(ctx context.Context, id string) (*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSecurityGroup", ctx, id)
	ret0, _ := ret[0].(*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSecurityGroup indicates an expected call of GetSecurityGroup.
func (mr *MockActorMockRecorder) GetSecurityGroup(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSecurityGroup", reflect.TypeOf((*MockActor)(nil).GetSecurityGroup), ctx, id)
}

// GetVSwitch mocks base method.
func (m *MockActor) GetVSwitch(ctx context.Context, id string) (*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVSwitch", ctx, id)
	ret0, _ := ret[0].(*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVSwitch indicates an expected call of GetVSwitch.
func (mr *MockActorMockRecorder) GetVSwitch(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVSwitch", reflect.TypeOf((*MockActor)(nil).GetVSwitch), ctx, id)
}

// GetVpc mocks base method.
func (m *MockActor) GetVpc(ctx context.Context, id string) (*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVpc", ctx, id)
	ret0, _ := ret[0].(*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVpc indicates an expected call of GetVpc.
func (mr *MockActorMockRecorder) GetVpc(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVpc", reflect.TypeOf((*MockActor)(nil).GetVpc), ctx, id)
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
func (mr *MockActorMockRecorder) LisIpv6Gateways(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LisIpv6Gateways", reflect.TypeOf((*MockActor)(nil).LisIpv6Gateways), arg0, arg1)
}

// ListEIPs mocks base method.
func (m *MockActor) ListEIPs(ctx context.Context, ids []string) ([]*aliclient.EIP, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEIPs", ctx, ids)
	ret0, _ := ret[0].([]*aliclient.EIP)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEIPs indicates an expected call of ListEIPs.
func (mr *MockActorMockRecorder) ListEIPs(ctx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEIPs", reflect.TypeOf((*MockActor)(nil).ListEIPs), ctx, ids)
}

// ListEnhanhcedNatGatewayAvailableZones mocks base method.
func (m *MockActor) ListEnhanhcedNatGatewayAvailableZones(ctx context.Context, region string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEnhanhcedNatGatewayAvailableZones", ctx, region)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEnhanhcedNatGatewayAvailableZones indicates an expected call of ListEnhanhcedNatGatewayAvailableZones.
func (mr *MockActorMockRecorder) ListEnhanhcedNatGatewayAvailableZones(ctx, region any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEnhanhcedNatGatewayAvailableZones", reflect.TypeOf((*MockActor)(nil).ListEnhanhcedNatGatewayAvailableZones), ctx, region)
}

// ListNatGateways mocks base method.
func (m *MockActor) ListNatGateways(ctx context.Context, ids []string) ([]*aliclient.NatGateway, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListNatGateways", ctx, ids)
	ret0, _ := ret[0].([]*aliclient.NatGateway)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListNatGateways indicates an expected call of ListNatGateways.
func (mr *MockActorMockRecorder) ListNatGateways(ctx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListNatGateways", reflect.TypeOf((*MockActor)(nil).ListNatGateways), ctx, ids)
}

// ListSecurityGroups mocks base method.
func (m *MockActor) ListSecurityGroups(ctx context.Context, ids []string) ([]*aliclient.SecurityGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListSecurityGroups", ctx, ids)
	ret0, _ := ret[0].([]*aliclient.SecurityGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListSecurityGroups indicates an expected call of ListSecurityGroups.
func (mr *MockActorMockRecorder) ListSecurityGroups(ctx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListSecurityGroups", reflect.TypeOf((*MockActor)(nil).ListSecurityGroups), ctx, ids)
}

// ListVSwitches mocks base method.
func (m *MockActor) ListVSwitches(ctx context.Context, ids []string) ([]*aliclient.VSwitch, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVSwitches", ctx, ids)
	ret0, _ := ret[0].([]*aliclient.VSwitch)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVSwitches indicates an expected call of ListVSwitches.
func (mr *MockActorMockRecorder) ListVSwitches(ctx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVSwitches", reflect.TypeOf((*MockActor)(nil).ListVSwitches), ctx, ids)
}

// ListVpcs mocks base method.
func (m *MockActor) ListVpcs(ctx context.Context, ids []string) ([]*aliclient.VPC, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListVpcs", ctx, ids)
	ret0, _ := ret[0].([]*aliclient.VPC)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListVpcs indicates an expected call of ListVpcs.
func (mr *MockActorMockRecorder) ListVpcs(ctx, ids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListVpcs", reflect.TypeOf((*MockActor)(nil).ListVpcs), ctx, ids)
}

// ModifyEIP mocks base method.
func (m *MockActor) ModifyEIP(ctx context.Context, id string, eip *aliclient.EIP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModifyEIP", ctx, id, eip)
	ret0, _ := ret[0].(error)
	return ret0
}

// ModifyEIP indicates an expected call of ModifyEIP.
func (mr *MockActorMockRecorder) ModifyEIP(ctx, id, eip any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModifyEIP", reflect.TypeOf((*MockActor)(nil).ModifyEIP), ctx, id, eip)
}

// ModifyVpc mocks base method.
func (m *MockActor) ModifyVpc(arg0 context.Context, arg1 string, arg2 *aliclient.VPC) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModifyVpc", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// ModifyVpc indicates an expected call of ModifyVpc.
func (mr *MockActorMockRecorder) ModifyVpc(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModifyVpc", reflect.TypeOf((*MockActor)(nil).ModifyVpc), arg0, arg1, arg2)
}

// RevokeSecurityGroupRule mocks base method.
func (m *MockActor) RevokeSecurityGroupRule(ctx context.Context, sgId, ruleId, direction string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevokeSecurityGroupRule", ctx, sgId, ruleId, direction)
	ret0, _ := ret[0].(error)
	return ret0
}

// RevokeSecurityGroupRule indicates an expected call of RevokeSecurityGroupRule.
func (mr *MockActorMockRecorder) RevokeSecurityGroupRule(ctx, sgId, ruleId, direction any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevokeSecurityGroupRule", reflect.TypeOf((*MockActor)(nil).RevokeSecurityGroupRule), ctx, sgId, ruleId, direction)
}

// UnAssociateEIP mocks base method.
func (m *MockActor) UnAssociateEIP(ctx context.Context, eip *aliclient.EIP) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnAssociateEIP", ctx, eip)
	ret0, _ := ret[0].(error)
	return ret0
}

// UnAssociateEIP indicates an expected call of UnAssociateEIP.
func (mr *MockActorMockRecorder) UnAssociateEIP(ctx, eip any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnAssociateEIP", reflect.TypeOf((*MockActor)(nil).UnAssociateEIP), ctx, eip)
}

// MockFactory is a mock of Factory interface.
type MockFactory struct {
	ctrl     *gomock.Controller
	recorder *MockFactoryMockRecorder
	isgomock struct{}
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
func (m *MockFactory) NewActor(accessKeyID, secretAccessKey, region string) (aliclient.Actor, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewActor", accessKeyID, secretAccessKey, region)
	ret0, _ := ret[0].(aliclient.Actor)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewActor indicates an expected call of NewActor.
func (mr *MockFactoryMockRecorder) NewActor(accessKeyID, secretAccessKey, region any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewActor", reflect.TypeOf((*MockFactory)(nil).NewActor), accessKeyID, secretAccessKey, region)
}
