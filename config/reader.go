package config

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/coinbase/redisbetween/utils"
	"github.com/hashicorp/go-getter"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type Version int64

type DynamicConfig interface {
	Config() (*Config, Version)
	Stop()
}

type dynamicConfig struct {
	current *Config
	version Version
	stop    chan<- bool
	wg      *sync.WaitGroup
	sync.RWMutex
}

func Load(ctx context.Context, opts *Options) (DynamicConfig, error) {
	return newDynamicConfig(ctx, opts)
}

func newDynamicConfig(ctx context.Context, opts *Options) (*dynamicConfig, error) {
	log := ctx.Value(utils.CtxLogKey).(*zap.Logger)
	stop := make(chan bool)
	configUpdate := make(chan *Config)

	var wg sync.WaitGroup
	current, currentHash, err := readConfig(opts, log)

	if err != nil {
		return nil, err
	}

	dyn := &dynamicConfig{
		current: current,
		version: Version(time.Now().UnixMilli()),
		stop:    stop,
		wg:      &wg,
	}

	wg.Add(2)
	go func(update <-chan *Config, stop <-chan bool) {
		defer wg.Done()
		for {
			select {
			case c := <-update:
				func(d *dynamicConfig) {
					d.Lock()
					defer d.Unlock()
					d.current = c
					d.version = Version(time.Now().UnixMilli())
				}(dyn)
			case <-stop:
				return
			}
		}
	}(configUpdate, stop)

	go func(opts *Options, updateChannel chan<- *Config, stopChannel <-chan bool) {
		log := log.With(zap.String("process", "config_poller"))

		defer wg.Done()
		defer close(updateChannel)

		interval := opts.PollInterval
		lastHash := currentHash

		for {
			select {
			case <-time.After(interval):
				c, currentHash, err := readConfig(opts, log)
				if err != nil {
					log.Error("Failed to fetch config", zap.Error(err))
					continue
				}

				if lastHash != currentHash {
					updateChannel <- c
					lastHash = currentHash
				}
			case <-stopChannel:
				return
			}
		}

	}(opts, configUpdate, stop)

	return dyn, nil
}

func readConfig(opts *Options, _ *zap.Logger) (*Config, string, error) {
	f, err := ioutil.TempFile("", "*.json")
	if err != nil {
		return nil, "", err
	}

	err = f.Close()
	if err != nil {
		return nil, "", err
	}

	pwd, _ := os.Getwd()
	client := &getter.Client{
		Ctx:  context.Background(),
		Src:  opts.Url,
		Dst:  f.Name(),
		Pwd:  pwd,
		Mode: getter.ClientModeFile,
	}

	if err := client.Get(); err != nil {
		return nil, "", err
	}

	f, err = os.Open(f.Name())
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()
	if err != nil {
		return nil, "", err
	}

	body, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, "", err
	}

	hash := md5.Sum(body)

	cfg := Config{
		Pretty:       opts.Pretty,
		Statsd:       opts.Statsd,
		Level:        opts.Level,
		Url:          opts.Url,
		PollInterval: opts.PollInterval,
		Upstreams:    []*Upstream{},
		Listeners:    []*Listener{},
	}

	if err = json.Unmarshal(body, &cfg); err != nil {
		return nil, "", err
	}

	return &cfg, hex.EncodeToString(hash[:]), nil
}

func (d *dynamicConfig) Config() (*Config, Version) {
	d.RLock()
	defer d.RUnlock()
	return d.current, d.version
}

func (d *dynamicConfig) Stop() {
	close(d.stop)
	d.wg.Wait()
}
