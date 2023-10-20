//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package metric
package metric

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"

	"trpc-system/go-opentelemetry/pkg/protocol/opentelemetry-ext/proto/operation"
	"trpc-system/go-opentelemetry/sdk/metric/internal/registry"
	"trpc-system/go-opentelemetry/sdk/remote"
)

// Setup setup metrics
func Setup(opts ...SetupOption) error {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return SetupByConfig(*cfg)
}

var pushStopFunc func()
var defaultPusher *push.Pusher

// SetupByConfig setup by config
func SetupByConfig(cfg Config) error {
	if !cfg.Enabled {
		return nil
	}
	if len(cfg.ClientHistogramBuckets) != 0 {
		setClientHandledHistogramBuckets(cfg.ClientHistogramBuckets)
	}
	if len(cfg.ServerHistogramBuckets) != 0 {
		setServerHandledHistogramBuckets(cfg.ServerHistogramBuckets)
	}
	registerRPCServerCounter()
	registerRPCClientCounter()
	registerRPCHandledHistograms()
	enableClientStreamHistograms()
	if cfg.ServerOwner != "" {
		serverMetadata.WithLabelValues(cfg.ServerOwner, cfg.CmdbID).Set(1)
	}
	if cfg.Configurator != nil {
		cfg.Configurator.RegisterConfigApplyFunc(genConfigApplyFunc(cfg))
	}
	// Etcd registration
	if cfg.EnabledRegister {
		if len(cfg.RegistryEndpoints) == 0 {
			return errors.New("metric: registry endpoints nil")
		}
		if cfg.Instance.TenantID == "" {
			return errors.New("metric: tenant id nil")
		}
		if cfg.Instance.Addr == "" {
			return errors.New("metric: exporter addr nil")
		}
		if cfg.TTL == 0 {
			cfg.TTL = DefaultRegisterTTL
		}
		opts := []registry.EtcdOption{
			registry.WithTLS(newTLSConfig(cfg.TLSCert)),
		}
		reg := NewEtcdRegistry(cfg.RegistryEndpoints, cfg.Instance.TenantID, opts...)
		_, err := reg.Register(context.Background(), &cfg.Instance, cfg.TTL)
		return err
	}
	// prometheus push
	if cfg.PrometheusPush.Enabled {
		pusher := push.New(cfg.PrometheusPush.URL, cfg.PrometheusPush.Job).Gatherer(prometheus.DefaultGatherer)
		if cfg.PrometheusPush.UseBasicAuth {
			pusher.BasicAuth(cfg.PrometheusPush.Username, cfg.PrometheusPush.Password)
		}
		for name, value := range cfg.PrometheusPush.Grouping {
			pusher.Grouping(name, value)
		}
		if len(cfg.PrometheusPush.HTTPHeaders) > 0 {
			pusher.Client(newHTTPDoerWithHeaders(cfg.PrometheusPush.HTTPHeaders))
		}
		err := pusher.Push()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(context.Background())
		pushStopFunc = cancel
		defaultPusher = pusher
		if cfg.PrometheusPush.Interval > 0 {
			ticker := time.NewTicker(cfg.PrometheusPush.Interval)
			go func() {
				for {
					select {
					case <-ticker.C:
						_ = pusher.Push()
					case <-ctx.Done():
						return
					}
				}
			}()
		}
	}
	return nil
}

// DeletePrometheusPush send delete request to prometheus push gateway
func DeletePrometheusPush() error {
	if defaultPusher == nil {
		return nil
	}
	pushStopFunc()
	return defaultPusher.Delete()
}

type pushHTTPDoer struct {
	headers map[string]string
	client  *http.Client
}

func (p *pushHTTPDoer) Do(r *http.Request) (*http.Response, error) {
	for k, v := range p.headers {
		r.Header.Set(k, v)
	}
	return p.client.Do(r)
}

func newHTTPDoerWithHeaders(headers map[string]string) push.HTTPDoer {
	return &pushHTTPDoer{
		headers: headers,
		client:  &http.Client{},
	}
}

type SetupOption func(config *Config)

// WithEnabled .
func WithEnabled(enabled bool) SetupOption {
	return func(config *Config) {
		config.Enabled = enabled
	}
}

// WithEnabledRegister .
func WithEnabledRegister(enabled bool) SetupOption {
	return func(config *Config) {
		config.EnabledRegister = enabled
	}
}

// WithEnabledZPage .
func WithEnabledZPage(enabled bool) SetupOption {
	return func(config *Config) {
		config.EnabledZPage = enabled
	}
}

// WithInstance .
func WithInstance(ins *Instance) SetupOption {
	return func(config *Config) {
		config.Instance = *ins
	}
}

// WithTTL register ttl
func WithTTL(ttl time.Duration) SetupOption {
	return func(config *Config) {
		config.TTL = ttl
	}
}

// WithRegistryEndpoints registry endpoints addr
func WithRegistryEndpoints(endpoints []string) SetupOption {
	return func(config *Config) {
		config.RegistryEndpoints = endpoints
	}
}

// WithTLSCert .
func WithTLSCert(cert TLSCert) SetupOption {
	return func(config *Config) {
		config.TLSCert = cert
	}
}

// WithClientHistogramBuckets set client histogram buckets
func WithClientHistogramBuckets(buckets []float64) SetupOption {
	return func(config *Config) {
		config.ClientHistogramBuckets = buckets
	}
}

// WithServerHistogramBuckets set server histogram buckets
func WithServerHistogramBuckets(buckets []float64) SetupOption {
	return func(config *Config) {
		config.ServerHistogramBuckets = buckets
	}
}

// WithTSLCert set TSLCert
func WithTSLCert(cert TLSCert) SetupOption {
	return func(config *Config) {
		config.TLSCert = cert
	}
}

// WithConfigurator .
func WithConfigurator(configurator remote.Configurator) SetupOption {
	return func(config *Config) {
		config.Configurator = configurator
	}
}

func genConfigApplyFunc(cfg Config) remote.ConfigApplyFunc {
	return func(config *operation.Operation) error {
		// update sever owners
		if len(config.GetOwners()) > 0 {
			var owners []string
			for _, v := range config.GetOwners() {
				owners = append(owners, v.GetName())
			}
			updateServerMetadata(strings.Join(owners, ";"), cfg.CmdbID)
		}
		return nil
	}
}

// WithMetricsPrometheusPush prometheus push config
func WithMetricsPrometheusPush(p PrometheusPushConfig) SetupOption {
	return func(config *Config) {
		c := DefaultPrometheusPushConfig()
		if p.Enabled {
			c.Enabled = true
		}
		if p.URL != "" {
			c.URL = p.URL
		}
		if p.Job != "" {
			c.Job = p.Job
		}
		if p.Interval > 0 {
			c.Interval = p.Interval
		}
		if p.UseBasicAuth {
			c.UseBasicAuth = true
		}
		if p.Username != "" {
			c.Username = p.Username
		}
		if p.Password != "" {
			c.Password = p.Password
		}
		if len(p.Grouping) > 0 {
			c.Grouping = p.Grouping
		}
		if len(p.HTTPHeaders) > 0 {
			c.HTTPHeaders = p.HTTPHeaders
		}
		config.PrometheusPush = c
	}
}
