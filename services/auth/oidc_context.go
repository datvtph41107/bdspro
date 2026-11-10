package auth

import "context"

type oidcKey struct{}
type oidcCapability struct {
	client *OAuth2Client
	secure bool
}

// OAuth2 provider clients are owned by services/auth and bound per Server.
func WithOAuth2(ctx context.Context, client *OAuth2Client, secure bool) context.Context {
	if ctx==nil {panic("auth: nil request context")}
	if client==nil {return ctx}
	return context.WithValue(ctx,oidcKey{},oidcCapability{client:client,secure:secure})
}
func OAuth2FromContext(ctx context.Context) (*OAuth2Client,bool) {
	if ctx==nil {return nil,false}
	owner,ok:=ctx.Value(oidcKey{}).(oidcCapability)
	if !ok {return nil,false}
	return owner.client,owner.secure
}
