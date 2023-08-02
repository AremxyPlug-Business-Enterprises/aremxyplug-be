# logger

## Description
Wrapper for zap logger package. More details in https://fcmbuk.atlassian.net/wiki/spaces/ROAV/pages/643072031/Logging

## Usage
* Default Configuration
```go
import (
	"github.com/roava/zebra/logger"
	"go.uber.org/zap"
)

log := logger.New()
log.Debug("debug")
log.Info("info")
log.Error("debug", zap.Error(errors.New("err")))

// Output
// {"level":"INFO","time":"2020-12-22T16:53:42.906-0300","msg":"info"}
// {"level":"ERROR","time":"2020-12-22T16:53:42.907-0300","msg":"debug","error":"err"}
```

* With name
```go
log = logger.New(logger.Config{
	Name: "test",
})
log.Debug("debug")
log.Info("info")
log.Error("debug", zap.Error(errors.New("err")))

// Output
{"level":"INFO","time":"2020-12-22T16:53:42.907-0300","logger":"test","msg":"info"}
{"level":"ERROR","time":"2020-12-22T16:53:42.907-0300","logger":"test","msg":"debug","error":"err"}
```

* With calling function's file name and line number
```go
log = logger.New(logger.Config{
	WithCaller: true,
})
log.Debug("debug")
log.Info("info")
log.Error("debug", zap.Error(errors.New("err")))

// Output
// {"level":"INFO","time":"2020-12-22T16:53:42.907-0300","caller":"zebra/main.go:34","msg":"info"}
// {"level":"ERROR","time":"2020-12-22T16:53:42.907-0300","caller":"zebra/main.go:35","msg":"debug","error":"err"}
```

* With stacktrace (only for Errors)
```go
log = logger.New(logger.Config{
	WithStacktrace: true,
})
log.Debug("debug")
log.Info("info")
log.Error("debug", zap.Error(errors.New("err")))

// Output
// {"level":"INFO","time":"2020-12-22T16:53:42.907-0300","msg":"info"}
// {"level":"ERROR","time":"2020-12-22T16:53:42.907-0300","msg":"debug","error":"err","stacktrace":"main.main\n\t/Users/rafael/go/src/github.com/roava/zebra/main.go:44\nruntime.main\n\t/usr/local/go/src/runtime/proc.go:204"}
```

* Changing log level
```go
log = logger.New(logger.Config{
	Level: zap.DebugLevel,
})
log.Debug("debug")
log.Info("info")
log.Error("debug", zap.Error(errors.New("err")))

// Output
// {"level":"DEBUG","time":"2020-12-22T16:53:42.907-0300","msg":"debug"}
// {"level":"INFO","time":"2020-12-22T16:53:42.907-0300","msg":"info"}
// {"level":"ERROR","time":"2020-12-22T16:53:42.907-0300","msg":"debug","error":"err"}
```

* With fields
```go
log = logger.New()
log.Info("user",
    zap.String("name", "Abc"),
    zap.Int("age", 30),
    zap.Strings("address", []string{"St", "Abc", "US"}))

// Output
// {"level":"INFO","time":"2020-12-22T19:27:42.032-0300","msg":"user","name":"Abc","age":30,"address":["St","Abc","US"]}
```

* Creating customizable logs
```go
type User struct {
	Name     string
	Password string
	Email    string
}

func (u *User) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", u.Name)
	enc.AddString("email", u.Email)
	return nil
}

user := &User{
	Name:     "John",
	Password: "1234",
	Email:    "john@gmail.com",
}

log = logger.New()
log.Info("user", zap.Object("user", user))

// Output
// {"level":"INFO","time":"2020-12-22T19:34:27.531-0300","msg":"user","user":{"name":"John","email":"john@gmail.com"}}
```

* With multiple outputs (files + stderr)
```go
log = logger.New(logger.Config{
	OutputPaths: []string{"logs", "test/logs", logger.Stderr},
})
log.Info("info")

// Output (stderr and in each output file)
// {"level":"INFO","time":"2020-12-22T19:34:27.531-0300","msg":"info"}
```

* Enabling DEBUG mode
```go
log = logger.New(logger.Config{
	Level:      zap.DebugLevel,
	WithCaller: true,
	Debug:      true,
})
log.Debug("debug")
log.Info("info")
log.Error("debug", zap.Error(errors.New("err")))

// Output
// 2020-12-22T16:53:42.907-0300    [DEBUG] zebra/main.go:62        debug
// 2020-12-22T16:53:42.907-0300    [INFO]  zebra/main.go:63        info
// 2020-12-22T16:53:42.907-0300    [ERROR] zebra/main.go:64        debug   {"error": "err"}
```
