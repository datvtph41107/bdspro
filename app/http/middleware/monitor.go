package middleware
import (
 "net/http"
 "time"
 "github.com/gin-gonic/gin"
 "github.com/go-redis/redis/v8"
 "github.com/rs/zerolog/log"
)
const securityDetectionThreshold int64=12
const securityDetectionWindow=10*time.Minute
var securityCountScript = redis.NewScript(`
local v = redis.call('INCR', KEYS[1])
if v == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return v
`)
func (l *RateLimiter) RequestMonitor()gin.HandlerFunc{
 return func(c *gin.Context){
  c.Next()
  status:=c.Writer.Status()
  reason,ok:=map[int]string{http.StatusBadRequest:"invalid_request",http.StatusUnauthorized:"authentication",http.StatusForbidden:"authorization",http.StatusTooManyRequests:"rate_limit",http.StatusRequestEntityTooLarge:"request_size"}[status]
  if !ok{return}
  path:=c.FullPath();if path==""{path="<unmatched>"}
  source:=rateKey(c.ClientIP())
  log.Ctx(c.Request.Context()).Warn().Str("reason",reason).Str("route",path).Str("method",c.Request.Method).Str("source_id",source[:16]).Int("status",status).Msg("security.request_denied")
  if l.redis==nil{return}
  n,err:=securityCountScript.Run(c.Request.Context(),l.redis,[]string{"bdspro:security:denied:v1:"+source},int64(securityDetectionWindow.Seconds())).Int64()
  if err!=nil{log.Ctx(c.Request.Context()).Warn().Str("component","request_monitor").Msg("security.monitor_unavailable");return}
  if n==securityDetectionThreshold{log.Ctx(c.Request.Context()).Warn().Str("reason","repeated_denials").Str("source_id",source[:16]).Int64("count",n).Dur("window",securityDetectionWindow).Msg("security.suspicious_activity")}
 }
}
