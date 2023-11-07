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

// Package remote
package remote

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/protocol/opentelemetry-ext/proto/operation"
)

// ConfigApplyFunc ...
type ConfigApplyFunc func(config *operation.Operation) error

// Configurator called when config changed
type Configurator interface {
	RegisterConfigApplyFunc(fn ConfigApplyFunc)
}

type remoteConfigurator struct {
	remoteServiceAddr string
	syncInterval      time.Duration
	tenantID          string
	app               string
	server            string
	debug             bool

	client              operation.OperationServiceClient
	lastConfig          *operation.Operation
	configApplyFuncList []ConfigApplyFunc
	// mu Protect lastConfig/configApplyFuncList.
	mu sync.Mutex
}

// NewRemoteConfigurator create a new remoteConfigurator
func NewRemoteConfigurator(remoteServiceAddr string, syncInterval time.Duration,
	tenantID, app, server string) Configurator {
	if syncInterval == 0 {
		syncInterval = time.Minute
	}
	rc := &remoteConfigurator{
		remoteServiceAddr: remoteServiceAddr,
		syncInterval:      syncInterval,
		tenantID:          tenantID,
		app:               app,
		server:            server,
	}
	// export OTEL_TRACE=remote
	if otelTraceEnv := os.Getenv("OTEL_TRACE"); strings.Contains(otelTraceEnv, "remote") {
		log.Printf("opentelemetry: env OTEL_TRACE:%s", otelTraceEnv)
		rc.debug = true
	}
	if rc.remoteServiceAddr != "" {
		go rc.syncDaemon()
	}
	return rc
}

func (rc *remoteConfigurator) syncDaemon() {
	for {
		rc.sync()
		time.Sleep(rc.syncInterval)
	}
}

func (rc *remoteConfigurator) sync() {
	if rc.client == nil {
		cc, err := grpc.Dial(rc.remoteServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			if rc.debug {
				log.Printf("opentelemetry: remote dial err:%v", err)
			}
			return
		}
		rc.client = operation.NewOperationServiceClient(cc)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{
		"x-tps-tenantid": rc.tenantID,
	}))
	req := &operation.GetOperationRequest{
		Tenant: rc.tenantID,
		App:    rc.app,
		Server: rc.server,
	}
	rsp, err := rc.client.GetOperation(ctx, req, grpc.WaitForReady(true))
	if err != nil {
		if rc.debug {
			log.Printf("opentelemetry: remote GetOperation err:%v", err)
		}
		return
	}
	if rc.debug {
		log.Printf("opentelemetry: remote GetOperation result:%+v", rsp)
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.lastConfig = rsp.GetOperation()
	for _, v := range rc.configApplyFuncList {
		if err = v(rsp.GetOperation()); err != nil {
			if rc.debug {
				log.Printf("opentelemetry: remote apply err:%v", err)
			}
		}
	}
}

// RegisterConfigApplyFunc register config change handler
func (rc *remoteConfigurator) RegisterConfigApplyFunc(fn ConfigApplyFunc) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.configApplyFuncList = append(rc.configApplyFuncList, fn)
	// Apply on register for async setup.
	if rc.lastConfig != nil {
		if err := fn(rc.lastConfig); err != nil && rc.debug {
			log.Printf("opentelemetry: remote apply err:%v", err)
		}
	}
}
