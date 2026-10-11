package organization

import (
 "encoding/json"
 "errors"
 "net/http"

 "github.com/datvtph41107/bdspro/auth"
)
type Handler struct{Service *Service; Auth *auth.Handler}
func (h *Handler) current(w http.ResponseWriter,r *http.Request)(int64,bool){
 id,ok:=h.Auth.Account(r)
 if !ok{http.Error(w,"unauthorized",401)}
 return id,ok
}
func inputJSON(w http.ResponseWriter,r *http.Request,dst any)bool{
 r.Body=http.MaxBytesReader(w,r.Body,16<<10)
 dec:=json.NewDecoder(r.Body);dec.DisallowUnknownFields()
 if err:=dec.Decode(dst);err!=nil{http.Error(w,"invalid request body",400);return false}
 return true
}
func respond(w http.ResponseWriter,status int,value any){
 w.Header().Set("Content-Type","application/json")
 w.Header().Set("Cache-Control","no-store")
 w.WriteHeader(status);_ = json.NewEncoder(w).Encode(value)
}
func handleErr(w http.ResponseWriter,err error){
 switch {
 case errors.Is(err,ErrForbidden),errors.Is(err,ErrNotFound):
  http.Error(w,"resource unavailable",404)
 case errors.Is(err,ErrInvalid):
  http.Error(w,"invalid request",400)
 case errors.Is(err,ErrConflict):
  http.Error(w,"conflict",409)
 default:
  http.Error(w,"internal server error",500)
 }
}
func (h *Handler) RegisterRoutes(mux *http.ServeMux){
 mux.HandleFunc("POST /organizations/registrations",func(w http.ResponseWriter,r *http.Request){
  actor,ok:=h.current(w,r);if !ok{return}
  var body struct{ Name string `json:"name"`; Jurisdiction string `json:"jurisdiction"`; RegistrationCode string `json:"registration_code"` }
  if !inputJSON(w,r,&body){return}
  id,err:=h.Service.SubmitRegistration(r.Context(),actor,body.Name,body.Jurisdiction,body.RegistrationCode)
  if err!=nil{handleErr(w,err);return}
  respond(w,201,map[string]any{"request_id":id,"status":"PENDING"})
 })
 mux.HandleFunc("POST /organizations/{id}/claims",func(w http.ResponseWriter,r *http.Request){
  actor,ok:=h.current(w,r);if !ok{return}
  org,err:=auth.PositiveID(r.PathValue("id"));if err!=nil{http.Error(w,"invalid organization id",400);return}
  var body struct{EvidenceReference string `json:"evidence_reference"`}
  if !inputJSON(w,r,&body){return}
  id,err:=h.Service.SubmitClaim(r.Context(),actor,org,body.EvidenceReference)
  if err!=nil{handleErr(w,err);return}
  respond(w,201,map[string]any{"request_id":id,"status":"PENDING"})
 })
 mux.HandleFunc("POST /platform/organizations/registrations/{id}/approve",func(w http.ResponseWriter,r *http.Request){
  reviewer,ok:=h.current(w,r);if !ok{return}
  id,err:=auth.PositiveID(r.PathValue("id"));if err!=nil{http.Error(w,"invalid request id",400);return}
  var body struct{ReviewReference string `json:"review_reference"`}
  if !inputJSON(w,r,&body){return}
  org,err:=h.Service.ApproveRegistration(r.Context(),reviewer,id,body.ReviewReference)
  if err!=nil{handleErr(w,err);return}
  respond(w,200,map[string]any{"organization_id":org,"status":"APPROVED"})
 })
 mux.HandleFunc("POST /platform/organizations/claims/{id}/approve",func(w http.ResponseWriter,r *http.Request){
  reviewer,ok:=h.current(w,r);if !ok{return}
  id,err:=auth.PositiveID(r.PathValue("id"));if err!=nil{http.Error(w,"invalid request id",400);return}
  var body struct{ReviewReference string `json:"review_reference"`}
  if !inputJSON(w,r,&body){return}
  org,err:=h.Service.ApproveInitialClaim(r.Context(),reviewer,id,body.ReviewReference)
  if err!=nil{handleErr(w,err);return}
  respond(w,200,map[string]any{"organization_id":org,"status":"APPROVED"})
 })
 mux.HandleFunc("POST /organizations/{id}/invitations",func(w http.ResponseWriter,r *http.Request){
  actor,ok:=h.current(w,r);if !ok{return}
  org,err:=auth.PositiveID(r.PathValue("id"));if err!=nil{http.Error(w,"invalid org id",400);return}
  var body struct{Email string `json:"email"`;RoleID int64 `json:"role_id"`}
  if !inputJSON(w,r,&body){return}
  id,err:=h.Service.Invite(r.Context(),actor,org,body.RoleID,body.Email)
  if err!=nil{handleErr(w,err);return}
  respond(w,201,map[string]any{"invitation_id":id,"status":"PENDING"})
 })
 mux.HandleFunc("POST /organizations/invitations/{id}/accept",func(w http.ResponseWriter,r *http.Request){
  actor,ok:=h.current(w,r);if !ok{return}
  id,err:=auth.PositiveID(r.PathValue("id"));if err!=nil{http.Error(w,"invalid invitation id",400);return}
  membership,err:=h.Service.AcceptInvitation(r.Context(),actor,id)
  if err!=nil{handleErr(w,err);return}
  respond(w,200,map[string]any{"membership_id":membership,"status":"ACTIVE"})
 })
 mux.HandleFunc("GET /organizations/{id}/members",func(w http.ResponseWriter,r *http.Request){
  actor,ok:=h.current(w,r);if !ok{return}
  org,err:=auth.PositiveID(r.PathValue("id"));if err!=nil{http.Error(w,"invalid org id",400);return}
  members,err:=h.Service.ListMembers(r.Context(),actor,org)
  if err!=nil{handleErr(w,err);return}
  respond(w,200,members)
 })
}
