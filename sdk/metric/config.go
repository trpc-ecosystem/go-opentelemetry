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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	"trpc-ecosystem/go-opentelemetry/sdk/remote"
)

// Config metric config
type Config struct {
	// Enabled open or close
	Enabled bool `yaml:"enabled"`
	// EnabledRegister default enabled
	EnabledRegister bool `yaml:"enable_register"`
	// RegistryEndpoints registry addrs
	RegistryEndpoints []string `yaml:"registry_endpoints"`
	// TLS credentials
	TLSCert TLSCert `yaml:"tls_cert"`
	// TTL Time to live
	TTL time.Duration `yaml:"ttl"`
	// Instance registry instance info
	Instance Instance `yaml:"instance"`
	// ServerOwner is server owner user, for send alert
	ServerOwner string `yaml:"server_owner"`
	// CmdbID is cmdbID
	CmdbID string `yaml:"cmdb_id"`
	// Codes code mapping
	Codes []*Code `yaml:"codes"`
	// Configurator only for experimental usage
	Configurator remote.Configurator `yaml:"-"`
	// ClientHistogramBuckets user can config client histogram buckets
	ClientHistogramBuckets []float64 `yaml:"client_histogram_buckets"`
	// ServerHistogramBuckets user can config server histogram buckets
	ServerHistogramBuckets []float64 `yaml:"server_histogram_buckets"`
	// ClientStreamRecvHistogramBuckets user can config client stream recv histogram buckets
	ClientStreamRecvHistogramBuckets []float64 `yaml:"client_stream_recv_histogram_buckets"`
	// ClientStreamSendHistogramBuckets user can config client stream recv histogram buckets
	ClientStreamSendHistogramBuckets []float64 `yaml:"client_stream_send_histogram_buckets"`
	// PrometheusPush prometheus push config
	PrometheusPush PrometheusPushConfig `yaml:"prometheus_push"`
	// EnabledZPage zPage option
	EnabledZPage bool
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		EnabledRegister: true,
	}
}

// DefaultPrometheusPushConfig default prometheus push config
func DefaultPrometheusPushConfig() PrometheusPushConfig {
	return PrometheusPushConfig{
		Job:      "reporter",
		Interval: time.Minute,
	}
}

const (
	instanceKeyPrefix = "/opentelemetry/metrics/services"
)

// TLSCert etcd need 3 credentials
type TLSCert struct {
	CertContent   string `yaml:"cert_content"`    // CA in string
	KeyContent    string `yaml:"key_content"`     // private key in string
	CaCertContent string `yaml:"ca_cert_content"` // CaCert in string
}

// newTLSConfig .
func newTLSConfig(tlsCert TLSCert) *tls.Config {
	// new TLS config from string
	if len(tlsCert.CertContent) > 0 && len(tlsCert.KeyContent) > 0 && len(tlsCert.CaCertContent) > 0 {
		cert, err := tls.X509KeyPair([]byte(tlsCert.CertContent), []byte(tlsCert.KeyContent))
		if err != nil {
			return nil
		}
		caCert := []byte(tlsCert.CaCertContent)
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			return nil
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		}
		return tlsConfig
	}
	return nil
}

// Instance ...
type Instance struct {
	// Addr metrics server
	Addr string `json:"addr" yaml:"addr"`
	// TenantID tenantID
	TenantID string `json:"tenant_id" yaml:"tenant_id"`
	// Metadata ...
	Metadata map[string]string `json:"metadata" yaml:"metadata"`
	// KeySuffix key suffix for Etcd registration
	KeySuffix string `json:"-" yaml:"-"`
	// Key user-defined key
	Key string `json:"-" yaml:"-"`
}

// GetKey ...
func (ins *Instance) GetKey() string {
	if ins == nil {
		return ""
	}
	if ins.Key != "" {
		return ins.Key
	}
	return fmt.Sprintf("%s/%s/%s%s", instanceKeyPrefix, ins.TenantID, ins.Addr, ins.KeySuffix)
}

// GetValue marshal instance to string
func (ins *Instance) GetValue() string {
	if ins != nil {
		if data, err := json.Marshal(ins); nil == err {
			return string(data)
		}
	}
	return "{}"
}

// PrometheusPushConfig prometheus push config
type PrometheusPushConfig struct {
	Enabled      bool              `yaml:"enabled"`
	URL          string            `yaml:"url"`
	Job          string            `yaml:"job"`
	Interval     time.Duration     `yaml:"interval"`
	UseBasicAuth bool              `yaml:"use_basic_auth"`
	Username     string            `yaml:"username"`
	Password     string            `yaml:"password"`
	Grouping     map[string]string `yaml:"grouping"`
	HTTPHeaders  map[string]string `yaml:"http_headers"`
}
