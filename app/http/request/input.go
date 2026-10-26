package request
import ("errors";"net/http";"strconv";"github.com/gin-gonic/gin")
func JSON(c *gin.Context,target any)bool{return check(c,c.ShouldBindJSON(target),"invalid request")}
func Query(c *gin.Context,target any,message string)bool{return check(c,c.ShouldBindQuery(target),message)}
func check(c *gin.Context,err error,msg string)bool{
 if err==nil{return true}
 var max *http.MaxBytesError
 if errors.As(err,&max){c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge,gin.H{"error":"request body too large"});return false}
 c.AbortWithStatusJSON(http.StatusBadRequest,gin.H{"error":msg});return false
}
func PositiveID(c *gin.Context,name string)(int64,bool){id,err:=strconv.ParseInt(c.Param(name),10,64);return id,err==nil&&id>0}
