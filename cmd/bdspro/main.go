package main

import (
    "context"
    "fmt"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    httpserver "github.com/datvtph41107/bdspro/app/http/server"
    database "github.com/datvtph41107/bdspro/db"
    db "github.com/datvtph41107/bdspro/db/sqlc"
    "github.com/datvtph41107/bdspro/mail"
    "github.com/datvtph41107/bdspro/util"
    "github.com/datvtph41107/bdspro/worker"
    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/hibiken/asynq"
    "github.com/rs/zerolog/log"
)

// main owns the process: signal handling, ordered construction, cleanup.
// HTTP, PostgreSQL, Redis and jobs own their implementation in other packages.
func main(){
    ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM)
    defer stop()
    if err:=run(ctx);err!=nil{log.Fatal().Err(err).Msg("BDSPro stopped")}
}

func run(ctx context.Context)error{
    started:=time.Now()
    config,err:=util.LoadConfig(".")
    if err!=nil{return fmt.Errorf("load config: %w",err)}
    log.Logger=util.NewLogger(os.Stderr,config.LogFormat)
    if os.Getenv(gin.EnvGinMode)==""{gin.SetMode(gin.ReleaseMode)}
    if config.LogFormat=="console"{fmt.Fprint(os.Stderr,"\nBDS PRO / API\n\n")}
    log.Info().Str("environment",config.Environment).Msg("startup.begin")

    pool,err:=database.Open(ctx,config.DBSource,config.MigrationURL)
    if err!=nil{return err}
    defer pool.Close()
    store:=db.NewStore(pool)

    if config.RedisAddress==""{return fmt.Errorf("REDIS_ADDRESS is required")}
    redisClient:=redis.NewClient(&redis.Options{Addr:config.RedisAddress})
    defer redisClient.Close()
    pingCtx,cancel:=context.WithTimeout(ctx,5*time.Second)
    err=redisClient.Ping(pingCtx).Err();cancel()
    if err!=nil{return fmt.Errorf("redis ping: %w",err)}
    log.Info().Msg("redis.connected")
    redisOpt:=asynq.RedisClientOpt{Addr:config.RedisAddress}
    distributor:=worker.NewRedisTaskDistributor(redisOpt)
    defer distributor.Close()

    mailer:=mail.NewSMTPSender(config.EmailSenderName,config.EmailSenderAddress,
        config.EmailSenderPassword,config.SMTPAddress,config.SMTPAuth)
    processor:=worker.NewRedisTaskProcessor(redisOpt,store,mailer,config.VerifyEmailURL)
    if err:=processor.Start();err!=nil{return fmt.Errorf("start jobs: %w",err)}
    defer processor.Shutdown()
    log.Info().Msg("task.processor.started")

    server,err:=httpserver.NewServer(config,store,distributor,redisClient)
    if err!=nil{return err}
    return server.StartWithReady(ctx,config.HTTPServerAddress,func(_ net.Addr){
        log.Info().Dur("duration",time.Since(started)).Msg("startup.ready")
    })
}
