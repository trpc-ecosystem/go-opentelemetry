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

package registry

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	// DefaultRegisterTTL default register ttl
	DefaultRegisterTTL = time.Second * 60
	// DefaultDialTimeout default dail timeout
	DefaultDialTimeout = time.Second * 5
)

// etcdRegistry register or unregister instance to etcd
type etcdRegistry struct {
	cli         *clientv3.Client
	tenantID    string
	registerTTL time.Duration
	err         error
}

// EtcdOption etcd config option
type EtcdOption func(*clientv3.Config)

// WithTLS TLS certs
func WithTLS(tlsConfig *tls.Config) EtcdOption {
	return func(config *clientv3.Config) {
		config.TLS = tlsConfig
	}
}

// NewEtcdRegistry new etcd registry
func NewEtcdRegistry(etcdEndpoints []string, tenantID string, opts ...EtcdOption) Registry {
	r := &etcdRegistry{
		registerTTL: DefaultRegisterTTL,
		tenantID:    tenantID,
	}
	cfg := clientv3.Config{
		Endpoints: etcdEndpoints,
		DialOptions: []grpc.DialOption{
			grpc.WithChainStreamInterceptor(r.streamInterceptor()),
			grpc.WithUnaryInterceptor(r.unaryInterceptor()),
		},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	r.cli, r.err = clientv3.New(cfg)

	return r
}

// Register register instance to etcd
func (e *etcdRegistry) Register(ctx context.Context, ins Instance, ttl time.Duration) (context.CancelFunc, error) {
	if err := e.err; err != nil {
		return nil, err
	}
	cctx, cancel := context.WithCancel(ctx)
	leaseID, err := e.register(cctx, ins, ttl)
	if err != nil {
		cancel()
		return nil, err
	}
	ch := make(chan struct{}, 1)
	cancelFunc := func() {
		cancel()
		<-ch
	}
	go func() {
		leaseID := leaseID
		for {
			err := e.keepAlive(cctx, leaseID)
			select {
			case <-cctx.Done():
				_ = e.unregister(context.Background(), ins)
				ch <- struct{}{}
				return
			default:
			}
			if err != nil {
				retryWait := e.registerTTL/3 + time.Duration(rand.Int63n(int64(e.registerTTL/3)))
				log.Printf("[E]opetelemetry: keepAlive error:%v, will retry after %v", err, retryWait)
				time.Sleep(retryWait)
				leaseID, _ = e.register(cctx, ins, ttl)
			}
		}
	}()
	return cancelFunc, nil
}

func (e *etcdRegistry) register(ctx context.Context, ins Instance, ttl time.Duration) (clientv3.LeaseID, error) {
	if err := e.err; err != nil {
		return 0, err
	}
	ttlResp, err := e.cli.Grant(ctx, int64(ttl.Seconds()))
	if err != nil {
		return 0, err
	}
	_, err = e.cli.Put(ctx, ins.GetKey(), ins.GetValue(), clientv3.WithLease(ttlResp.ID))
	if err != nil {
		return 0, err
	}
	return ttlResp.ID, nil
}

// keepAlive Blocking until the context is canceled or the underlying connection is disconnected for a long time
// exceeding TTL, ch will be closed, and the caller needs to re-initiate the lease.
func (e *etcdRegistry) keepAlive(
	ctx context.Context, id clientv3.LeaseID) error {
	ch, err := e.cli.KeepAlive(ctx, id)
	if err != nil {
		return fmt.Errorf("keepAlive err:%w", err)
	}
	for range ch {
	} // noop
	return fmt.Errorf("keepAlive ch closed")
}

func (e *etcdRegistry) unregister(ctx context.Context, ins Instance) error {
	if err := e.err; err != nil {
		return err
	}
	if _, err := e.cli.Delete(ctx, ins.GetKey()); err != nil {
		return err
	}
	return nil
}

func (*etcdRegistry) contextWithTenantID(ctx context.Context, tenantID string) context.Context {
	const tenantHeaderKey = "x-tps-tenantid" // the header of grpc is in lowercase
	return metadata.AppendToOutgoingContext(ctx, tenantHeaderKey, tenantID)
}

func (e *etcdRegistry) unaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(e.contextWithTenantID(ctx, e.tenantID), method, req, reply, cc, opts...)
	}
}

func (e *etcdRegistry) streamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer,
		opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return streamer(e.contextWithTenantID(ctx, e.tenantID), desc, cc, method, opts...)
	}
}
