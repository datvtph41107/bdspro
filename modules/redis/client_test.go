package redis

import (
    "context"
    "testing"
)
func TestConnectRejectsMissingRedisAddress(t *testing.T) {
    if _,err:=Connect(context.Background(),"");err==nil {t.Fatal("missing Redis address accepted")}
}
