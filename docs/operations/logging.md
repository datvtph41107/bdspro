# Logging lifecycle

`modules/log` owns logger manager, synchronous console/file sinks, formatted
log API and structured event access. The system uses Zerolog under that API,
not Gitea's exact async writer manager.

```text
main.go
  -> cmd.PrepareConsoleLoggerLevel (early safe stderr)
  -> cmd.loadSettings
     -> setting.InitLoggers (configured sinks/levels)
  -> web or worker
  -> log.GetManager().Close (release owned file writers)
```

In local `app.ini` set `[log] MODE = console`; output goes to stderr.
For file output set `MODE = file` and a writable `ROOT_PATH`, plus
`[log.file] FILE_NAME`. Docker setup prepares `/data/bdspro/log`.
There is no reason to commit a root `logs/` folder or generated log files.

`Log.Info("...",args...)` provides formatted diagnostics, while
`log.Ctx(ctx)` provides structured fields for request/task correlation.
Never log raw OTP codes, verification secrets, JWTs, HTTP Authorization,
passwords, or SMTP credentials. Log safe event names, request route templates,
status, durations and failure types instead.

Availability and performance tradeoff: synchronous output can block under
slow writers. An async queue/rotation package is justified only after
measuring contention and defining loss/flush semantics.
