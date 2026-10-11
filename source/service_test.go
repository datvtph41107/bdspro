package source

import (
 "bytes"
 "context"
 "encoding/json"
 "errors"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "sync"
 "testing"

 "github.com/jackc/pgx/v5/pgconn"
 "github.com/jackc/pgx/v5/pgxpool"
)

type fixtures struct {db *pgxpool.Pool; service *Service; orgA,orgB,alice,bob int64}

func testDB(t *testing.T)*pgxpool.Pool {
 t.Helper()
 url:=os.Getenv("SOURCE_TEST_DATABASE_URL")
 if url==""{t.Skip("requires SOURCE_TEST_DATABASE_URL")}
 db,err:=pgxpool.New(context.Background(),url)
 if err!=nil{t.Fatal(err)}
 t.Cleanup(db.Close)
 var name string
 if err:=db.QueryRow(context.Background(),"SELECT current_database()").Scan(&name);err!=nil{t.Fatal(err)}
 if name!="bdspro_source_ci"{t.Fatalf("refusing destructive fixture setup outside bdspro_source_ci, got %q",name)}
 if _,err:=db.Exec(context.Background(),"TRUNCATE TABLE parties, permissions RESTART IDENTITY CASCADE");err!=nil{t.Fatal(err)}
 mustExec(t,db,"INSERT INTO permissions(code) VALUES ('sources.create'),('sources.read'),('sources.price_report')")
 return db
}
func mustID(t *testing.T,db *pgxpool.Pool,query string,args ...any)int64{
 t.Helper()
 var id int64
 if err:=db.QueryRow(context.Background(),query,args...).Scan(&id);err!=nil{t.Fatal(err)}
 return id
}
func mustExec(t *testing.T,db *pgxpool.Pool,query string,args ...any){
 t.Helper()
 if _,err:=db.Exec(context.Background(),query,args...);err!=nil{t.Fatal(err)}
}
func newOrg(t *testing.T,db *pgxpool.Pool,name string)int64{
 ctx:=context.Background()
 tx,err:=db.Begin(ctx);if err!=nil{t.Fatal(err)}
 defer tx.Rollback(ctx)
 var id int64
 if err:=tx.QueryRow(ctx,"INSERT INTO parties(kind) VALUES('ORGANIZATION') RETURNING id").Scan(&id);err!=nil{t.Fatal(err)}
 if _,err:=tx.Exec(ctx,"INSERT INTO organizations(party_id,name) VALUES($1,$2)",id,name);err!=nil{t.Fatal(err)}
 if err:=tx.Commit(ctx);err!=nil{t.Fatal(err)}
 return id
}
func newAccount(t *testing.T,db *pgxpool.Pool)int64{
 ctx:=context.Background()
 tx,err:=db.Begin(ctx);if err!=nil{t.Fatal(err)}
 defer tx.Rollback(ctx)
 var person,account int64
 if err:=tx.QueryRow(ctx,"INSERT INTO parties(kind) VALUES('PERSON') RETURNING id").Scan(&person);err!=nil{t.Fatal(err)}
 if _,err:=tx.Exec(ctx,"INSERT INTO persons(party_id) VALUES($1)",person);err!=nil{t.Fatal(err)}
 if err:=tx.QueryRow(ctx,"INSERT INTO accounts(person_id) VALUES($1) RETURNING id",person).Scan(&account);err!=nil{t.Fatal(err)}
 if err:=tx.Commit(ctx);err!=nil{t.Fatal(err)}
 return account
}
func grant(t *testing.T,db *pgxpool.Pool,orgID,accountID int64,perms ...string){
 var person int64
 if err:=db.QueryRow(context.Background(),"SELECT person_id FROM accounts WHERE id=$1",accountID).Scan(&person);err!=nil{t.Fatal(err)}
 member:=mustID(t,db,"INSERT INTO organization_memberships(organization_id,person_id) VALUES($1,$2) RETURNING id",orgID,person)
 role:=mustID(t,db,"INSERT INTO organization_roles(organization_id,name) VALUES($1,$2) RETURNING id",orgID,fmt.Sprintf("role-%d",accountID))
 mustExec(t,db,"INSERT INTO organization_role_assignments(organization_id,membership_id,role_id) VALUES($1,$2,$3)",orgID,member,role)
 for _,p:=range perms{mustExec(t,db,"INSERT INTO organization_role_permissions(role_id,permission_code) VALUES($1,$2)",role,p)}
}
func setup(t *testing.T,perms ...string)fixtures{
 db:=testDB(t)
 f:=fixtures{db:db,service:NewService(db)}
 f.orgA=newOrg(t,db,"An Phu"); f.orgB=newOrg(t,db,"Nam Viet")
 f.alice=newAccount(t,db); f.bob=newAccount(t,db)
 grant(t,db,f.orgA,f.alice,perms...)
 grant(t,db,f.orgB,f.bob,"sources.create","sources.read","sources.price_report")
 return f
}
func sample()CaptureInput{
 return CaptureInput{Channel:"zalo",Label:"house",AreaM2:"75",InitialPrice:&PriceInput{Amount:"4200000000",Currency:"VND"}}
}
func count(t *testing.T,db *pgxpool.Pool,table string)int{
 t.Helper()
 if table!="source_intakes"&&table!="source_price_reports"{t.Fatal("invalid count table")}
 var n int
 if err:=db.QueryRow(context.Background(),"SELECT count(*) FROM "+table).Scan(&n);err!=nil{t.Fatal(err)}
 return n
}

func TestCaptureAndHistory(t *testing.T){
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 ctx:=context.Background()
 c,err:=f.service.Capture(ctx,f.alice,f.orgA,"first",sample())
 if err!=nil||c.ID<=0||c.Replayed{t.Fatalf("%+v %v",c,err)}
 src,err:=f.service.Read(ctx,f.alice,f.orgA,c.ID)
 if err!=nil||src.Channel!="zalo"||src.AreaM2==""{t.Fatalf("%+v %v",src,err)}
 _,err=f.service.ReportPrice(ctx,f.alice,f.orgA,c.ID,"second",PriceInput{Amount:"4100000000",Currency:"VND"})
 if err!=nil{t.Fatal(err)}
 history,err:=f.service.PriceHistory(ctx,f.alice,f.orgA,c.ID)
 if err!=nil||len(history)!=2{t.Fatalf("%+v %v",history,err)}
 if history[0].Amount!="4200000000.0000"||history[1].Amount!="4100000000.0000"{t.Fatalf("history ordering/data: %+v",history)}
}

func TestCaptureIdempotencyAndPriceRetry(t *testing.T){
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 ctx:=context.Background()
 first,err:=f.service.Capture(ctx,f.alice,f.orgA,"retry",sample());if err!=nil{t.Fatal(err)}
 again,err:=f.service.Capture(ctx,f.alice,f.orgA,"retry",sample())
 if err!=nil||!again.Replayed||first.ID!=again.ID{t.Fatalf("%+v %v",again,err)}
 changed:=sample();changed.InitialPrice.Amount="4300000000"
 if _,err=f.service.Capture(ctx,f.alice,f.orgA,"retry",changed);!errors.Is(err,ErrKeyConflict){t.Fatalf("expected key conflict: %v",err)}
 price:=PriceInput{Amount:"4100000000.0",Currency:"vnd"}
 p,err:=f.service.ReportPrice(ctx,f.alice,f.orgA,first.ID,"price1",price);if err!=nil{t.Fatal(err)}
 pr,err:=f.service.ReportPrice(ctx,f.alice,f.orgA,first.ID,"price1",price)
 if err!=nil||!pr.Replayed||pr.ID!=p.ID{t.Fatalf("%+v %v",pr,err)}
 price.Amount="4000000000"
 if _,err=f.service.ReportPrice(ctx,f.alice,f.orgA,first.ID,"price1",price);!errors.Is(err,ErrKeyConflict){t.Fatalf("expected price conflict: %v",err)}
 if count(t,f.db,"source_intakes")!=1||count(t,f.db,"source_price_reports")!=2{t.Fatal("duplicate rows")}
}
func TestCrossTenantAndRevocation(t *testing.T){
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 ctx:=context.Background()
 c,err:=f.service.Capture(ctx,f.alice,f.orgA,"tenant",sample());if err!=nil{t.Fatal(err)}
 if _,err=f.service.Read(ctx,f.bob,f.orgA,c.ID);!errors.Is(err,ErrNotFound){t.Fatalf("cross org read: %v",err)}
 if _,err=f.service.ReportPrice(ctx,f.bob,f.orgB,c.ID,"wrong",PriceInput{Amount:"1",Currency:"VND"});!errors.Is(err,ErrForbidden){t.Fatalf("cross org write: %v",err)}
 if _,err=f.service.Capture(ctx,f.bob,f.orgA,"wrong",sample());!errors.Is(err,ErrForbidden){t.Fatalf("cross org capture: %v",err)}
 mustExec(t,f.db,"UPDATE organization_memberships SET ended_at=now() WHERE organization_id=$1",f.orgA)
 if _,err=f.service.Read(ctx,f.alice,f.orgA,c.ID);!errors.Is(err,ErrNotFound){t.Fatalf("revoked read: %v",err)}
 if _,err=f.service.ReportPrice(ctx,f.alice,f.orgA,c.ID,"revoked",PriceInput{Amount:"1",Currency:"VND"});!errors.Is(err,ErrForbidden){t.Fatalf("revoked write: %v",err)}
}
func TestCaptureAtomicity(t *testing.T){
 f:=setup(t,"sources.create","sources.read")
 if _,err:=f.service.Capture(context.Background(),f.alice,f.orgA,"atomic",sample());!errors.Is(err,ErrForbidden){t.Fatalf("must deny initial price: %v",err)}
 if count(t,f.db,"source_intakes")!=0||count(t,f.db,"source_price_reports")!=0{t.Fatal("orphan capture after denied price")}
 input:=sample();input.InitialPrice=nil
 if _,err:=f.service.Capture(context.Background(),f.alice,f.orgA,"without-price",input);err!=nil{t.Fatal(err)}
}
func TestConcurrentCaptureSameKey(t *testing.T){
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 var wg sync.WaitGroup
 ids:=make(chan int64,5);errs:=make(chan error,5)
 for i:=0;i<5;i++{wg.Add(1);go func(){defer wg.Done();c,e:=f.service.Capture(context.Background(),f.alice,f.orgA,"concurrent",sample());ids<-c.ID;errs<-e}()}
 wg.Wait();close(ids);close(errs)
 for e:=range errs{if e!=nil{t.Fatal(e)}}
 var id int64
 for got:=range ids{if id==0{id=got};if got<=0||got!=id{t.Fatalf("duplicate IDs %d/%d",id,got)}}
 if count(t,f.db,"source_intakes")!=1||count(t,f.db,"source_price_reports")!=1{t.Fatal("concurrent duplicates")}
}
func TestDatabaseConstraints(t *testing.T){
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 c,err:=f.service.Capture(context.Background(),f.alice,f.orgA,"fk",sample());if err!=nil{t.Fatal(err)}
 sql:="INSERT INTO source_price_reports(organization_id,intake_id,recorded_by_account_id,amount,currency_code) VALUES($1,$2,$3,$4,$5)"
 _,err=f.db.Exec(context.Background(),sql,f.orgB,c.ID,f.alice,100,"VND")
 var p *pgconn.PgError
 if !errors.As(err,&p)||p.Code!="23503"{t.Fatalf("expected FK 23503, got %v",err)}
 _,err=f.db.Exec(context.Background(),sql,f.orgA,c.ID,f.alice,-1,"VND")
 if !errors.As(err,&p)||p.Code!="23514"{t.Fatalf("expected CHECK 23514, got %v",err)}
}
func TestHTTPWithInjectedVerifiedPrincipal(t *testing.T){
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 handler:=NewHandler(f.service,func(*http.Request)(int64,bool){return f.alice,true})
 url:=fmt.Sprintf("/organizations/%d/sources",f.orgA)
 payload:=[]byte("{\"channel\":\"zalo\",\"area_m2\":\"75\",\"initial_price\":{\"amount\":\"4200000000\",\"currency\":\"VND\"}}")
 request:=func(h http.Handler,method,url,key string,body []byte)*httptest.ResponseRecorder{
  r:=httptest.NewRequest(method,url,bytes.NewReader(body));if key!=""{r.Header.Set("Idempotency-Key",key)}
  w:=httptest.NewRecorder();h.ServeHTTP(w,r);return w
 }
 first:=request(handler,"POST",url,"http-key",payload)
 if first.Code!=201{t.Fatalf("create %d: %s",first.Code,first.Body.String())}
 var body struct{ID int64;Replayed bool}
 if err:=json.Unmarshal(first.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
 replay:=request(handler,"POST",url,"http-key",payload)
 if replay.Code!=200{t.Fatalf("retry %d: %s",replay.Code,replay.Body.String())}
 read:=request(handler,"GET",fmt.Sprintf("%s/%d",url,body.ID),"",nil)
 if read.Code!=200{t.Fatalf("read %d: %s",read.Code,read.Body.String())}
 none:=request(NewHandler(f.service,nil),"GET",fmt.Sprintf("%s/%d",url,body.ID),"",nil)
 if none.Code!=401{t.Fatalf("missing session must be 401, got %d",none.Code)}
}
func TestNormalizeDecimal(t *testing.T){
 got,err:=normalizeDecimal("0004.2000",priceRe)
 if err!=nil||got!="4.2"{t.Fatalf("got=%q err=%v",got,err)}
 for _,v:=range []string{"0","-1","1.00001","1e9","NaN",""}{
  if _,err:=normalizeDecimal(v,priceRe);err==nil{t.Fatalf("accepted %q",v)}
 }
}


func TestMembershipBlockAndClosedAccount(t *testing.T) {
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 ctx:=context.Background()
 c,err:=f.service.Capture(ctx,f.alice,f.orgA,"block",sample())
 if err!=nil{t.Fatal(err)}
 var membership int64
 if err:=f.db.QueryRow(ctx,"SELECT id FROM organization_memberships WHERE organization_id=$1",f.orgA).Scan(&membership);err!=nil{t.Fatal(err)}
 block:=mustID(t,f.db,"INSERT INTO organization_membership_blocks(membership_id) VALUES($1) RETURNING id",membership)
 if _,err:=f.service.Read(ctx,f.alice,f.orgA,c.ID);!errors.Is(err,ErrNotFound){t.Fatalf("blocked read: %v",err)}
 mustExec(t,f.db,"UPDATE organization_membership_blocks SET ended_at=now() WHERE id=$1",block)
 if _,err:=f.service.Read(ctx,f.alice,f.orgA,c.ID);err!=nil{t.Fatalf("after unblock: %v",err)}
 mustExec(t,f.db,"UPDATE accounts SET closed_at=now() WHERE id=$1",f.alice)
 if _,err:=f.service.Read(ctx,f.alice,f.orgA,c.ID);!errors.Is(err,ErrNotFound){t.Fatalf("closed account read: %v",err)}
}
func TestPriceStatementAndSourceReceptionTimes(t *testing.T) {
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 ctx:=context.Background()
 in:=sample()
 in.ReceivedAt="2026-10-09T12:00:00+07:00"
 in.InitialPrice.StatementAt="2026-10-09T11:30:00+07:00"
 c,err:=f.service.Capture(ctx,f.alice,f.orgA,"times",in)
 if err!=nil{t.Fatal(err)}
 src,err:=f.service.Read(ctx,f.alice,f.orgA,c.ID)
 if err!=nil || src.ReceivedAt==nil {t.Fatalf("received at: %+v err=%v",src,err)}
 history,err:=f.service.PriceHistory(ctx,f.alice,f.orgA,c.ID)
 if err!=nil || len(history)!=1 || history[0].StatementAt==nil {t.Fatalf("statement time: %+v err=%v",history,err)}
 if history[0].StatementAt.After(history[0].RecordedAt) {t.Fatal("source statement unexpectedly after DB recording")}
}
func TestConcurrentPriceReportRetry(t *testing.T) {
 f:=setup(t,"sources.create","sources.read","sources.price_report")
 ctx:=context.Background()
 in:=sample();in.InitialPrice=nil
 c,err:=f.service.Capture(ctx,f.alice,f.orgA,"capture-for-price",in)
 if err!=nil{t.Fatal(err)}
 var wg sync.WaitGroup
 ids:=make(chan int64,4);errs:=make(chan error,4)
 for i:=0;i<4;i++{wg.Add(1);go func(){
  defer wg.Done()
  p,e:=f.service.ReportPrice(ctx,f.alice,f.orgA,c.ID,"same-price",
   PriceInput{Amount:"4100000000",Currency:"VND"})
  ids<-p.ID;errs<-e
 }()}
 wg.Wait();close(ids);close(errs)
 for e:=range errs{if e!=nil{t.Fatalf("price race: %v",e)}}
 var expected int64
 for got:=range ids{if expected==0{expected=got};if got<=0||got!=expected{t.Fatalf("price ids %d/%d",got,expected)}}
 if count(t,f.db,"source_price_reports")!=1{t.Fatal("duplicate price on retry race")}
}
