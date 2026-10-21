package gapi

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/organization"
	pb "github.com/datvtph41107/bdspro/pb/bdspro/v1"
	"github.com/datvtph41107/bdspro/token"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type sessionFake struct {
	id int64
	err error
	calls int
}
func (s *sessionFake) AccountFromSessionHash(_ context.Context, hash string) (int64, error) {
	s.calls++
	if len(hash) != 64 {
		return 0, auth.ErrUnauthorized
	}
	return s.id, s.err
}

type workspacesFake struct {
	items []organization.Workspace
	err error
	seen int64
}
func (s *workspacesFake) ListWorkspaces(_ context.Context, actor int64) ([]organization.Workspace, error) {
	s.seen = actor
	return s.items, s.err
}

func rpcClient(t *testing.T, sessions *sessionFake, workspaces *workspacesFake) (pb.AccountServiceClient, *token.JWTMaker) {
	t.Helper()
	maker, err := token.NewJWTMaker(strings.Repeat("G", 32))
	if err != nil { t.Fatal(err) }
	srv, err := NewServer(maker, sessions, workspaces)
	if err != nil { t.Fatal(err) }
	listener := bufconn.Listen(1024 * 1024)
	t.Cleanup(func() { srv.Stop(); listener.Close() })
	go func() { _ = srv.Serve(listener) }()
	dialCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { t.Fatal(err) }
	t.Cleanup(func() { conn.Close() })
	// NewClient is lazy; use one bounded context for the first RPC in tests.
	_ = dialCtx
	return pb.NewAccountServiceClient(conn), maker
}

func bearer(ctx context.Context, value string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", value))
}

func TestRPCJWTSessionAndWorkspaceProjection(t *testing.T) {
	joined := time.Date(2026,10,21,2,30,0,0,time.UTC)
	sessions := &sessionFake{id:81}
	workspaces := &workspacesFake{items:[]organization.Workspace{{
		OrganizationID: 99, OrganizationName: "A", MembershipID: 12, JoinedAt: joined,
	}}}
	client, maker := rpcClient(t, sessions, workspaces)
	access, _, err := maker.Issue(81, "real-session-secret")
	if err != nil { t.Fatal(err) }
	ctx, cancel := context.WithTimeout(bearer(context.Background(), "Bearer "+access), 3*time.Second)
	defer cancel()
	acct, err := client.GetCurrentAccount(ctx, &pb.GetCurrentAccountRequest{})
	if err != nil || acct.GetAccountId()!=81 || acct.GetUser().GetAccountId()!=81 {
		t.Fatalf("account lookup: %+v %v", acct,err)
	}
	ws, err := client.ListWorkspaces(ctx, &pb.ListWorkspacesRequest{})
	if err!=nil || len(ws.GetWorkspaces())!=1 || ws.GetWorkspaces()[0].GetOrganizationId()!=99 ||
		!ws.GetWorkspaces()[0].GetJoinedAt().AsTime().Equal(joined) || workspaces.seen!=81 {
		t.Fatalf("workspace: %+v %v",ws,err)
	}
	sessions.err = auth.ErrUnauthorized
	_, err = client.GetCurrentAccount(ctx, &pb.GetCurrentAccountRequest{})
	if status.Code(err)!=codes.Unauthenticated { t.Fatalf("revoked session passed: %v",err) }
}

func TestRPCBearerCannotFallBackOrForgeIdentity(t *testing.T) {
	sessions:=&sessionFake{id:7}
	client,maker:=rpcClient(t,sessions,&workspacesFake{})
	good,_,err:=maker.Issue(7,"session")
	if err!=nil {t.Fatal(err)}
	wrong,_,err:=maker.Issue(33,"session")
	if err!=nil {t.Fatal(err)}
	for _,tc:= range []struct{name string; ctx context.Context}{
		{"no token",context.Background()},
		{"untrusted cookie",metadata.NewOutgoingContext(context.Background(),metadata.Pairs("cookie","bdspro_session=valid"))},
		{"malformed bearer",bearer(context.Background(),"Bearer invalid")},
		{"wrong jwt subject",bearer(context.Background(),"Bearer "+wrong)},
		{"duplicate authorization",metadata.NewOutgoingContext(context.Background(),metadata.Pairs("authorization","Bearer "+good,"authorization","Bearer "+good))},
	}{
		t.Run(tc.name,func(t *testing.T){
			ctx,cancel:=context.WithTimeout(tc.ctx,3*time.Second); defer cancel()
			_,err:=client.GetCurrentAccount(ctx,&pb.GetCurrentAccountRequest{})
			if status.Code(err)!=codes.Unauthenticated {t.Fatalf("expected Unauthenticated: %v",err)}
		})
	}
}

func TestRPCDependencyFailureDoesNotLeakError(t *testing.T) {
	sessions:=&sessionFake{id:7,err:errors.New("sensitive pg error")}
	ws:=&workspacesFake{}
	client,maker:=rpcClient(t,sessions,ws)
	raw,_,err:=maker.Issue(7,"session")
	if err!=nil {t.Fatal(err)}
	ctx,cancel:=context.WithTimeout(bearer(context.Background(),"Bearer "+raw),3*time.Second);defer cancel()
	_,err=client.GetCurrentAccount(ctx,&pb.GetCurrentAccountRequest{})
	if status.Code(err)!=codes.Unavailable || strings.Contains(err.Error(),"sensitive") {t.Fatalf("leaked pg err: %v",err)}
	sessions.err=nil
	ws.err=errors.New("private workspace failure")
	_,err=client.ListWorkspaces(ctx,&pb.ListWorkspacesRequest{})
	if status.Code(err)!=codes.Unavailable || strings.Contains(err.Error(),"private") {t.Fatalf("workspace error: %v",err)}
}

func TestRPCServerRejectsMissingDependencies(t *testing.T) {
	maker,_:=token.NewJWTMaker(strings.Repeat("G",32))
	if _,err:=NewServer(nil,&sessionFake{},&workspacesFake{});err==nil {t.Fatal("nil maker accepted")}
	if _,err:=NewServer(maker,nil,&workspacesFake{});err==nil {t.Fatal("nil auth accepted")}
	if _,err:=NewServer(maker,&sessionFake{},nil);err==nil {t.Fatal("nil workspaces accepted")}
}
