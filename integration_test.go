package main

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "testing"

 "github.com/datvtph41107/bdspro/auth"
 "github.com/datvtph41107/bdspro/organization"
 "github.com/datvtph41107/bdspro/source"
 "github.com/jackc/pgx/v5"
 "github.com/jackc/pgx/v5/pgconn"
 "github.com/jackc/pgx/v5/pgxpool"
)

func coreDB(t *testing.T)*pgxpool.Pool{
 t.Helper()
 url:=os.Getenv("CORE_TEST_DATABASE_URL")
 if url==""{t.Skip("requires CORE_TEST_DATABASE_URL")}
 ctx:=context.Background()
 db,err:=pgxpool.New(ctx,url);if err!=nil{t.Fatal(err)}
 t.Cleanup(db.Close)
 var name string
 if err:=db.QueryRow(ctx,"SELECT current_database()").Scan(&name);err!=nil{t.Fatal(err)}
 if name!="bdspro_core_ci"{t.Fatalf("refusing to truncate %q (requires bdspro_core_ci)",name)}
 _,err=db.Exec(ctx,"TRUNCATE parties,permissions,platform_organization_reviewers RESTART IDENTITY CASCADE")
 if err!=nil{t.Fatal(err)}
 _,err=db.Exec(ctx,"INSERT INTO permissions(code) VALUES('members.read'),('members.invite'),('sources.create'),('sources.read'),('sources.price_report')")
 if err!=nil{t.Fatal(err)}
 return db
}

func coreUser(t *testing.T,service *auth.Service,email string)(int64,string){
 t.Helper()
 ctx:=context.Background()
 reg,err:=service.Register(ctx,email,"ComplexPassphrase123!")
 if err!=nil{t.Fatal(err)}
 if err:=service.VerifyEmail(ctx,reg.VerificationToken);err!=nil{t.Fatal(err)}
 login,err:=service.Login(ctx,email,"ComplexPassphrase123!")
 if err!=nil{t.Fatal(err)}
 if reg.AccountID!=login.AccountID||login.Token==""{t.Fatal("wrong login identity")}
 return login.AccountID,login.Token
}

func TestCoreJourneyRegisterOrganizationInviteSource(t *testing.T){
 db:=coreDB(t)
 ctx:=context.Background()
 a:=auth.New(db,nil)
 org:=organization.New(db)
 sources:=source.NewService(db)

 reviewer,_:=coreUser(t,a,"reviewer@example.test")
 an,anToken:=coreUser(t,a,"an@example.test")
 lan,lanToken:=coreUser(t,a,"lan@example.test")
 _,err:=db.Exec(ctx,`INSERT INTO platform_organization_reviewers(account_id,authorization_reference)
 VALUES ($1,'operator-approved-test-fixture')`,reviewer)
 if err!=nil{t.Fatal(err)}

 reg,err:=org.SubmitRegistration(ctx,an,"ABC Realty","VN","AB123")
 if err!=nil{t.Fatal(err)}
 // Applicant cannot claim Organization administrator rights before review.
 var organizationsBefore int
 if err:=db.QueryRow(ctx,"SELECT count(*) FROM organizations").Scan(&organizationsBefore);err!=nil{t.Fatal(err)}
 if organizationsBefore!=0{t.Fatalf("unexpected org before approval: %d",organizationsBefore)}
 if _,err:=org.ApproveRegistration(ctx,an,reg,"self-approved");!errors.Is(err,organization.ErrForbidden){
  t.Fatalf("applicant approved self: %v",err)
 }
 abc,err:=org.ApproveRegistration(ctx,reviewer,reg,"operator-verified-registry")
 if err!=nil{t.Fatal(err)}
 if abc<=0{t.Fatal("no company")}
 if _,err:=org.ApproveRegistration(ctx,reviewer,reg,"repeat");!errors.Is(err,organization.ErrNotFound){
  t.Fatalf("duplicate reviewer action: %v",err)
 }
 var role int64
 if err:=db.QueryRow(ctx,`SELECT id FROM organization_roles
 WHERE organization_id=$1 AND name='Workspace Administrator'`,abc).Scan(&role);err!=nil{t.Fatal(err)}
 if role<=0{t.Fatal("no first admin role")}
 // The initial admin can invite; receiving an invitation creates no membership.
 invitation,err:=org.Invite(ctx,an,abc,role,"lan@example.test")
 if err!=nil{t.Fatal(err)}
 var count int
 if err:=db.QueryRow(ctx,"SELECT count(*) FROM organization_memberships WHERE organization_id=$1",abc).Scan(&count);err!=nil{t.Fatal(err)}
 if count!=1{t.Fatalf("invite granted early membership: %d",count)}
 membership,err:=org.AcceptInvitation(ctx,lan,invitation)
 if err!=nil{t.Fatal(err)}
 if membership<=0{t.Fatal("missing membership")}
 if _,err:=org.AcceptInvitation(ctx,lan,invitation);!errors.Is(err,organization.ErrNotFound){
  t.Fatalf("double acceptance: %v",err)
 }
 members,err:=org.ListMembers(ctx,lan,abc)
 if err!=nil||len(members)!=2{t.Fatalf("member list: %v %#v",err,members)}

 // A real Session identifies the same Account seen in SQL Authorization.
 tokenAccount,err:=a.AccountFromToken(ctx,lanToken)
 if err!=nil||tokenAccount!=lan{t.Fatalf("session actor: %d %v",tokenAccount,err)}
 s,err:=sources.Capture(ctx,lan,abc,"capture-from-lan",source.CaptureInput{
  Channel:"zalo",AreaM2:"75",
  InitialPrice:&source.PriceInput{Amount:"4200000000",Currency:"VND"},
 })
 if err!=nil{t.Fatal(err)}
 if s.ID<=0{t.Fatal("source not inserted")}
 _,err=sources.ReportPrice(ctx,an,abc,s.ID,"price-after-call",source.PriceInput{Amount:"4100000000",Currency:"VND"})
 if err!=nil{t.Fatal(err)}
 history,err:=sources.PriceHistory(ctx,lan,abc,s.ID)
 if err!=nil||len(history)!=2{t.Fatalf("history: %+v %v",history,err)}

 // A second organization must not accept ABC's role for a different tenant.
 reg2,err:=org.SubmitRegistration(ctx,reviewer,"XYZ Homes","VN","XYZ987")
 if err!=nil{t.Fatal(err)}
 xyz,err:=org.ApproveRegistration(ctx,reviewer,reg2,"verified XYZ")
 if err!=nil{t.Fatal(err)}
 if _,err:=sources.Read(ctx,an,xyz,s.ID);!errors.Is(err,source.ErrNotFound){
  t.Fatalf("cross-tenant read: %v",err)
 }
 if _,err:=org.ListMembers(ctx,lan,xyz);!errors.Is(err,organization.ErrNotFound){
  t.Fatalf("cross-tenant members: %v",err)
 }
 claim,err:=org.SubmitClaim(ctx,lan,abc,"pretend-to-represent-abc")
 if err!=nil{t.Fatal(err)}
 if _,err:=org.ApproveInitialClaim(ctx,reviewer,claim,"fake-claim");!errors.Is(err,organization.ErrConflict){
  t.Fatalf("existing admin was replaced: %v",err)
 }
 // End of tenancy revokes ability to read stored data.
 if _,err:=db.Exec(ctx,"UPDATE organization_memberships SET ended_at=now() WHERE id=$1",membership);err!=nil{t.Fatal(err)}
 if _,err:=sources.Read(ctx,lan,abc,s.ID);!errors.Is(err,source.ErrNotFound){t.Fatalf("revoked source: %v",err)}
 if err:=a.Logout(ctx,anToken);err!=nil{t.Fatal(err)}
 if _,err:=a.AccountFromToken(ctx,anToken);!errors.Is(err,auth.ErrUnauthorized){
  t.Fatalf("logout token replay: %v",err)
 }
}

func TestC01DeferredSubtypeAndRegistrationRollback(t *testing.T){
 db:=coreDB(t)
 ctx:=context.Background()
 tx,err:=db.Begin(ctx);if err!=nil{t.Fatal(err)}
 if _,err=tx.Exec(ctx,"INSERT INTO parties(kind) VALUES('PERSON')");err!=nil{t.Fatal(err)}
 err=tx.Commit(ctx)
 var pgErr *pgconn.PgError
 if !errors.As(err,&pgErr)||pgErr.Code!="23514"{
  t.Fatalf("expected C01 at COMMIT, got %v",err)
 }
 authService:=auth.New(db,nil)
 _,err=authService.Register(ctx,"duplicate@example.test","ValidLongPassword333!")
 if err!=nil{t.Fatal(err)}
 var before int
 if err:=db.QueryRow(ctx,"SELECT count(*) FROM parties").Scan(&before);err!=nil{t.Fatal(err)}
 _,err=authService.Register(ctx,"DUPLICATE@example.test","ValidLongPassword333!")
 if !errors.Is(err,auth.ErrEmailUsed){t.Fatalf("expected duplicate email: %v",err)}
 var after int
 if err:=db.QueryRow(ctx,"SELECT count(*) FROM parties").Scan(&after);err!=nil{t.Fatal(err)}
 if before!=after{t.Fatalf("registration left orphan party: before=%d after=%d",before,after)}
}

func TestAuthCookieLifecycle(t *testing.T){
 db:=coreDB(t)
 service:=auth.New(db,nil)
 h:=&auth.Handler{S:service,Development:true,SecureCookies:false}
 mux:=http.NewServeMux()
 h.RegisterRoutes(mux)
 act:=func(method,path string,body []byte,cookie *http.Cookie)*httptest.ResponseRecorder{
  req:=httptest.NewRequest(method,path,bytes.NewReader(body))
  if cookie!=nil{req.AddCookie(cookie)}
  w:=httptest.NewRecorder();mux.ServeHTTP(w,req);return w
 }
 payload:=[]byte("{"email":"minh@example.test","password":"VeryLongPassword1234!"}")
 registered:=act("POST","/auth/register",payload,nil)
 if registered.Code!=201{t.Fatalf("register HTTP: %d %s",registered.Code,registered.Body.String())}
 var reg struct{VerificationToken string `json:"verification_token"`}
 if err:=json.Unmarshal(registered.Body.Bytes(),&reg);err!=nil{t.Fatal(err)}
 v,_:=json.Marshal(map[string]string{"token":reg.VerificationToken})
 if r:=act("POST","/auth/verify-email",v,nil);r.Code!=204{t.Fatalf("verify HTTP: %d %s",r.Code,r.Body.String())}
 logged:=act("POST","/auth/login",payload,nil)
 if logged.Code!=200{t.Fatalf("login HTTP: %d %s",logged.Code,logged.Body.String())}
 cookies:=logged.Result().Cookies()
 if len(cookies)==0{t.Fatal("no session cookie")}
 me:=act("GET","/auth/me",nil,cookies[0])
 if me.Code!=200{t.Fatalf("me: %d %s",me.Code,me.Body.String())}
 logout:=act("POST","/auth/logout",nil,cookies[0])
 if logout.Code!=204{t.Fatalf("logout %d",logout.Code)}
 me=act("GET","/auth/me",nil,cookies[0])
 if me.Code!=401{t.Fatalf("replayed cookie after logout: %d",me.Code)}
}

func TestInvitationDoesNotCrossTenant(t *testing.T){
 db:=coreDB(t)
 ctx:=context.Background()
 a:=auth.New(db,nil)
 org:=organization.New(db)
 reviewer,_:=coreUser(t,a,"reviewer-scope@example.test")
 actor,_:=coreUser(t,a,"actor-scope@example.test")
 _,err:=db.Exec(ctx,"INSERT INTO platform_organization_reviewers(account_id,authorization_reference) VALUES($1,'reviewer-fixture')",reviewer)
 if err!=nil{t.Fatal(err)}
 regA,err:=org.SubmitRegistration(ctx,actor,"Scope-A","VN","SCOPEA");if err!=nil{t.Fatal(err)}
 abc,err:=org.ApproveRegistration(ctx,reviewer,regA,"proof");if err!=nil{t.Fatal(err)}
 regB,err:=org.SubmitRegistration(ctx,reviewer,"Scope-B","VN","SCOPEB");if err!=nil{t.Fatal(err)}
 xyz,err:=org.ApproveRegistration(ctx,reviewer,regB,"proof");if err!=nil{t.Fatal(err)}
 var roleB int64
 if err:=db.QueryRow(ctx,"SELECT id FROM organization_roles WHERE organization_id=$1",xyz).Scan(&roleB);err!=nil{t.Fatal(err)}
 if _,err:=org.Invite(ctx,actor,abc,roleB,"stranger@example.test");!errors.Is(err,organization.ErrForbidden){
  t.Fatalf("cross organization role grant: %v",err)
 }
 // Raw insert is also protected by composite FK.
 var memberA int64
 if err:=db.QueryRow(ctx,"SELECT id FROM organization_memberships WHERE organization_id=$1",abc).Scan(&memberA);err!=nil{t.Fatal(err)}
 _,err=db.Exec(ctx,"INSERT INTO organization_role_assignments(organization_id,membership_id,role_id) VALUES($1,$2,$3)",abc,memberA,roleB)
 var pgErr *pgconn.PgError
 if !errors.As(err,&pgErr)||pgErr.Code!="23503"{t.Fatalf("expected role-scope FK 23503: %v",err)}
}

func ExampleWorkflow() {
 fmt.Println("Account -> Session -> Organization review -> Invitation -> Membership -> Source history")
 // Output: Account -> Session -> Organization review -> Invitation -> Membership -> Source history
}
