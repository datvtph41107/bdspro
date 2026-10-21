package integration

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/datvtph41107/bdspro/api"
	"github.com/datvtph41107/bdspro/auth"
	"github.com/datvtph41107/bdspro/gapi"
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

func TestGRPCAndGinUseSamePostgresSessionAndWorkspaces(t *testing.T) {
	db:=coreDB(t)
	a:=auth.New(db,nil)
	o:=organization.New(db)
	accountID,opaque:=coreUser(t,a,"grpc-owner@example.test")
	reviewer,_:=coreUser(t,a,"grpc-reviewer@example.test")
	if _,err:=db.Exec(context.Background(),`INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
VALUES($1,'test-reviewed')`,reviewer);err!=nil {t.Fatal(err)}
	reg,err:=o.SubmitRegistration(context.Background(),accountID,"RPC Workspace","VN","RPC-01")
	if err!=nil {t.Fatal(err)}
	orgID,err:=o.ApproveRegistration(context.Background(),reviewer,reg,"review-ok")
	if err!=nil {t.Fatal(err)}
	maker,err:=token.NewJWTMaker(strings.Repeat("Q",32))
	if err!=nil {t.Fatal(err)}
	jwt,_,err:=maker.Issue(accountID,opaque)
	if err!=nil {t.Fatal(err)}

	server,err:=gapi.NewServer(maker,a,o)
	if err!=nil {t.Fatal(err)}
	lis:=bufconn.Listen(1024*1024)
	t.Cleanup(func(){server.Stop(); lis.Close()})
	go func(){_ = server.Serve(lis)}()
	conn,err:=grpc.NewClient("bufnet",grpc.WithContextDialer(func(context.Context,string)(net.Conn,error){
		return lis.Dial()
	}),grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err!=nil {t.Fatal(err)}
	t.Cleanup(func(){conn.Close()})
	client:=pb.NewAccountServiceClient(conn)
	ctx,cancel:=context.WithTimeout(metadata.NewOutgoingContext(context.Background(),metadata.Pairs("authorization","Bearer "+jwt)),5*time.Second)
	defer cancel()
	current,err:=client.GetCurrentAccount(ctx,&pb.GetCurrentAccountRequest{})
	if err!=nil || current.GetAccountId()!=accountID {t.Fatalf("gRPC actor=%+v err=%v",current,err)}
	grpcWorkspaces,err:=client.ListWorkspaces(ctx,&pb.ListWorkspacesRequest{})
	if err!=nil || len(grpcWorkspaces.GetWorkspaces())!=1 || grpcWorkspaces.Workspaces[0].GetOrganizationId()!=orgID {
		t.Fatalf("RPC memberships %+v %v",grpcWorkspaces,err)
	}

	httpAPI:=api.NewServer(db,a,o,nil,true,"")
	httpAPI.EnableJWTMaker(maker)
	req:=httptest.NewRequest(http.MethodGet,"/auth/workspaces",nil)
	req.Header.Set("Authorization","Bearer "+jwt)
	w:=httptest.NewRecorder()
	httpAPI.Handler().ServeHTTP(w,req)
	if w.Code!=200 {t.Fatalf("HTTP lookup status %d: %s",w.Code,w.Body.String())}
	var envelope struct{Workspaces []organization.Workspace `json:"workspaces"`}
	if err:=json.Unmarshal(w.Body.Bytes(),&envelope);err!=nil {t.Fatal(err)}
	if len(envelope.Workspaces)!=1 || envelope.Workspaces[0].OrganizationID!=grpcWorkspaces.Workspaces[0].GetOrganizationId() ||
		envelope.Workspaces[0].MembershipID!=grpcWorkspaces.Workspaces[0].GetMembershipId(){
		t.Fatalf("HTTP and gRPC mismatch: %+v vs %+v",envelope.Workspaces,grpcWorkspaces)
	}

	if err:=a.Logout(context.Background(),opaque);err!=nil {t.Fatal(err)}
	_,err=client.GetCurrentAccount(ctx,&pb.GetCurrentAccountRequest{})
	if status.Code(err)!=codes.Unauthenticated {t.Fatalf("revoked DB Session still accepted on gRPC: %v",err)}
	w=httptest.NewRecorder()
	httpAPI.Handler().ServeHTTP(w,req)
	if w.Code!=http.StatusUnauthorized {t.Fatalf("revoked DB Session still accepted on Gin: %d",w.Code)}
}
