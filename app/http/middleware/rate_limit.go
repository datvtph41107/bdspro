package middleware
import (
 "crypto/sha256"
 "encoding/hex"
 "fmt"
 "net/http"
 "strings"
 "time"
 "github.com/gin-gonic/gin"
 "github.com/go-redis/redis/v8"
 "github.com/rs/zerolog/log"
)
type RateLimiter struct { redis *redis.Client }
func NewRateLimiter(r *redis.Client)*RateLimiter{return &RateLimiter{redis:r}}
var authRateScript = redis.NewScript(`
local a = redis.call('INCR', KEYS[1])
if a == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
local b = redis.call('INCR', KEYS[2])
if b == 1 then redis.call('EXPIRE', KEYS[2], ARGV[1]) end
if a > tonumber(ARGV[2]) or b > tonumber(ARGV[3]) then return 0 end
return 1
`)
var ipActiveSourceScript = redis.NewScript(`
local n = redis.call('INCR', KEYS[1])
if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end
return n
`)
func rateKey(v string)string{ h:=sha256.Sum256([]byte(v));return hex.EncodeToString(h[:]) }
func (l *RateLimiter) Source(operation string,max int64,window time.Duration)gin.HandlerFunc{
 return func(c *gin.Context){
  if l.redis==nil {c.Next();return}
  key:="bdspro:source_limit:v1:"+operation+":"+rateKey(c.ClientIP())
  n,err:=ipActiveSourceScript.Run(c.Request.Context(),l.redis,[]string{key},int64(window.Seconds())).Int64()
  if err!=nil { log.Ctx(c.Request.Context()).Error().Str("operation",operation).Msg("auth.rate_limit_unavailable");c.AbortWithStatusJSON(http.StatusServiceUnavailable,gin.H{"error":"authentication temporarily unavailable"});return}
  if n>max {c.Header("Retry-After",fmt.Sprint(int64(window.Seconds())));c.AbortWithStatusJSON(http.StatusTooManyRequests,gin.H{"error":"too many requests"});return}
  c.Next()
 }
}
func (l *RateLimiter) Identity(c *gin.Context,operation,identity string,perIP,perIdentity int64,window time.Duration)bool{
 if l.redis==nil{return true}
 prefix:="bdspro:auth_limit:v1:"+operation+":"
 keys:=[]string{prefix+"ip:"+rateKey(c.ClientIP()),prefix+"identity:"+rateKey(strings.ToLower(strings.TrimSpace(identity)))}
 allowed,err:=authRateScript.Run(c.Request.Context(),l.redis,keys,int64(window.Seconds()),perIP,perIdentity).Int64()
 if err!=nil {log.Ctx(c.Request.Context()).Error().Err(err).Str("operation",operation).Msg("auth.rate_limit_unavailable");c.AbortWithStatusJSON(http.StatusServiceUnavailable,gin.H{"error":"authentication temporarily unavailable"});return false}
 if allowed!=1{c.Header("Retry-After",fmt.Sprint(int64(window.Seconds())));c.AbortWithStatusJSON(http.StatusTooManyRequests,gin.H{"error":"too many requests"});return false}
 return true
}
