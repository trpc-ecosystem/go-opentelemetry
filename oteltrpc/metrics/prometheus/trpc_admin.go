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

package prometheus

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gopkg.in/yaml.v2"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/log"

	"trpc-system/go-opentelemetry/api"
	oteladmin "trpc-system/go-opentelemetry/pkg/admin"
	"trpc-system/go-opentelemetry/sdk/metric"
)

// Setup .
func Setup(tenantID string, etcdEndpoints []string, opts ...metric.SetupOption) {
	initSink()
	admin.HandleFunc("/metrics", metric.LimitMetricsHandler().ServeHTTP)
	if tenantID == "" {
		tenantID = "default"
	}
	addr, needListen := getPrometheusServerAddr()
	log.Infof("opentelemetry: setup prometheus metrics server at http://%s/metrics", addr)
	if needListen {
		// admin port is not served, start http server
		cfg := metric.DefaultConfig()
		for _, opt := range opts {
			opt(cfg)
		}
		srv, err := oteladmin.NewServer(
			oteladmin.WithAddr(addr),
			oteladmin.WithEnablePprof(true),
			oteladmin.WithEnablePrometheus(true),
			oteladmin.WithEnableHotSwitch(true),
			oteladmin.WithEnableZPage(cfg.EnabledZPage),
		)
		if err != nil {
			log.Errorf("failed to new admin server: %v", err)
		}
		go func() {
			log.Infof("opentelemetry: start admin server because trpc admin is not served, addr: %s", addr)
			if err := srv.Serve(); err != nil && err != http.ErrServerClosed {
				log.Errorf("opentelemetry: failed to serve opentelemetry admin server: %v", err)
			}
		}()
	}
	cfg := trpc.GlobalConfig()
	containerName := cfg.Global.ContainerName
	if containerName == "" {
		containerName = addr
	}
	instance := &metric.Instance{
		TenantID: tenantID,
		Addr:     addr,
		Metadata: map[string]string{
			api.AppKey:           cfg.Server.App,
			api.ServerKey:        cfg.Server.Server,
			api.NamespaceKey:     cfg.Global.Namespace,
			api.EnvNameKey:       cfg.Global.EnvName,
			api.ContainerNameKey: containerName,
		},
	}
	go func() {
		setupOpts := []metric.SetupOption{
			metric.WithRegistryEndpoints(etcdEndpoints),
			metric.WithTTL(time.Second * 60),
			metric.WithInstance(instance),
		}
		setupOpts = append(setupOpts, opts...)
		err := metric.Setup(setupOpts...)
		if err != nil {
			log.Errorf("opentelemetry: metrics endpoint register etcd err:%v, endpoints:%v", err, etcdEndpoints)
			return
		}
	}()
}

func getPrometheusServerAddr() (addr string, shouldListen bool) {
	adminConfig := trpc.GlobalConfig().Server.Admin
	if adminConfig.Port > 0 {
		return fmt.Sprintf("%s:%d", adminConfig.IP, adminConfig.Port), false
	}
	if ip, port, err := getAdminAddrFromTRPCConfig(); err == nil {
		return fmt.Sprintf("%s:%d", ip, port), true
	}
	return "127.0.0.1:12621", true
}

func getAdminAddrFromTRPCConfig() (ip string, port uint16, err error) {
	buf, err := ioutil.ReadFile(trpc.ServerConfigPath)
	if err != nil {
		return "", 0, fmt.Errorf("read file err:%s", err)
	}
	cfg := &trpcConfig{}
	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return "", 0, fmt.Errorf("unmarshal file err:%s", err)
	}
	if cfg.Global.LocalIP != "" && cfg.Global.AdminPort > 0 {
		return cfg.Global.LocalIP, cfg.Global.AdminPort, nil
	}
	return "", 0, fmt.Errorf("invalid ip port:%s %d", cfg.Global.LocalIP, cfg.Global.AdminPort)
}

type trpcConfig struct {
	Global struct {
		LocalIP   string `yaml:"local_ip"`
		AdminPort uint16 `yaml:"admin_port"`
	}
}
