// Code generated for the Store interface of the Lecture 37 baseline. DO NOT EDIT.
// Regenerate with: make mock
package mockdb

import (
    "context"
    "reflect"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/google/uuid"
    "go.uber.org/mock/gomock"
)

type MockStore struct {
    ctrl *gomock.Controller
    recorder *MockStoreMockRecorder
}
type MockStoreMockRecorder struct { mock *MockStore }
func NewMockStore(ctrl *gomock.Controller) *MockStore {
    mock := &MockStore{ctrl:ctrl}
    mock.recorder = &MockStoreMockRecorder{mock}
    return mock
}
func (m *MockStore) EXPECT() *MockStoreMockRecorder { return m.recorder }

func (m *MockStore) CreatePersonParty(ctx context.Context) (int64, error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "CreatePersonParty", ctx)
    result0, _ := ret[0].(int64)
    result1, _ := ret[1].(error)
    return result0, result1
}
func (r *MockStoreMockRecorder) CreatePersonParty(arg0 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "CreatePersonParty", reflect.TypeOf((*MockStore)(nil).CreatePersonParty), arg0)
}

func (m *MockStore) CreatePerson(ctx context.Context, partyID int64) (error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "CreatePerson", ctx, partyID)
    result0, _ := ret[0].(error)
    return result0
}
func (r *MockStoreMockRecorder) CreatePerson(arg0 any, arg1 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "CreatePerson", reflect.TypeOf((*MockStore)(nil).CreatePerson), arg0, arg1)
}

func (m *MockStore) CreateAccount(ctx context.Context, arg db.CreateAccountParams) (db.Account, error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "CreateAccount", ctx, arg)
    result0, _ := ret[0].(db.Account)
    result1, _ := ret[1].(error)
    return result0, result1
}
func (r *MockStoreMockRecorder) CreateAccount(arg0 any, arg1 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "CreateAccount", reflect.TypeOf((*MockStore)(nil).CreateAccount), arg0, arg1)
}

func (m *MockStore) GetAccountByEmail(ctx context.Context, email string) (db.Account, error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "GetAccountByEmail", ctx, email)
    result0, _ := ret[0].(db.Account)
    result1, _ := ret[1].(error)
    return result0, result1
}
func (r *MockStoreMockRecorder) GetAccountByEmail(arg0 any, arg1 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "GetAccountByEmail", reflect.TypeOf((*MockStore)(nil).GetAccountByEmail), arg0, arg1)
}

func (m *MockStore) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "CreateSession", ctx, arg)
    result0, _ := ret[0].(db.Session)
    result1, _ := ret[1].(error)
    return result0, result1
}
func (r *MockStoreMockRecorder) CreateSession(arg0 any, arg1 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "CreateSession", reflect.TypeOf((*MockStore)(nil).CreateSession), arg0, arg1)
}

func (m *MockStore) GetSession(ctx context.Context, id uuid.UUID) (db.Session, error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "GetSession", ctx, id)
    result0, _ := ret[0].(db.Session)
    result1, _ := ret[1].(error)
    return result0, result1
}
func (r *MockStoreMockRecorder) GetSession(arg0 any, arg1 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "GetSession", reflect.TypeOf((*MockStore)(nil).GetSession), arg0, arg1)
}

func (m *MockStore) RegisterTx(ctx context.Context, arg db.RegisterTxParams) (db.Account, error) {
    m.ctrl.T.Helper()
    ret := m.ctrl.Call(m, "RegisterTx", ctx, arg)
    result0, _ := ret[0].(db.Account)
    result1, _ := ret[1].(error)
    return result0, result1
}
func (r *MockStoreMockRecorder) RegisterTx(arg0 any, arg1 any) *gomock.Call {
    r.mock.ctrl.T.Helper()
    return r.mock.ctrl.RecordCallWithMethodType(r.mock, "RegisterTx", reflect.TypeOf((*MockStore)(nil).RegisterTx), arg0, arg1)
}
