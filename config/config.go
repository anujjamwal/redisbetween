package config

import (
	"go.uber.org/zap/zapcore"
	"time"
)

type Config struct {
	Pretty    bool
	Statsd    string
	Level     zapcore.Level
	Listeners []*Listener
	Upstreams []*Upstream
}

type Mirroring struct {
	Target string
}

type Listener struct {
	Name              string
	Network           string
	LocalSocketPrefix string
	LocalSocketSuffix string
	Target            string
	MaxSubscriptions  int
	MaxBlockers       int
	Unlink            bool
	Mirroring         Mirroring
}

type Upstream struct {
	Name         string
	Address      string
	Database     int
	MaxPoolSize  int
	MinPoolSize  int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Readonly     bool
}
