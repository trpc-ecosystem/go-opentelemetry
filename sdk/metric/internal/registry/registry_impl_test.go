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
	"io"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
)

// Test_etcdRegistry_retry test etcd register with retry.
// 1. When the service starts to initiate registration, due to network problems or server processing timeout,
// use the retry Interceptor that comes with the etcd client to perform random backoff retries.
// https://github.com/etcd-io/etcd/blob/main/client/v3/retry_interceptor.go#L40
// 2. During the keepalive process, due to network problems or server processing timeout,
// retry is performed within the Register function. The retry interval is 1/3~2/3 TTL.
func Test_etcdRegistry_retry(t *testing.T) {
	addr := []string{"127.0.0.1:22379"}
	ctx := context.Background()
	ins := &testInstance{
		key:   "/a",
		value: "{}",
	}
	// async start etcd server
	var (
		s         *embed.Etcd
		etcdAddrs []string
	)
	var (
		proxyServer       net.Listener
		forceCloseAllFunc func()
		err               error
	)
	started := make(chan struct{})
	go func() {
		// start etcd server after start register
		time.Sleep(time.Millisecond * 200)
		s, etcdAddrs, _ = startSingleNodeETCD(t, "etcd1", 12380, 12390)
		// async start proxy server
		proxyServer, forceCloseAllFunc, err = simpleTCPProxy(addr[0], etcdAddrs[0])
		assert.NoError(t, err)
		close(started)
	}()
	var client *clientv3.Client
	go func() {
		<-started
		// watch all etcd event
		cfg := clientv3.Config{
			Endpoints: etcdAddrs,
		}
		client, err = clientv3.New(cfg)
		assert.NoError(t, err)
		for {
			c := client.Watch(ctx, "/", clientv3.WithPrefix())
			for v := range c {
				for _, vv := range v.Events {
					log.Printf("watch event:%+v", vv)
				}
			}
		}
	}()
	go func() {
		// start pprof
		_ = pprof.Index
		_ = http.ListenAndServe("127.0.0.1:12330", http.DefaultServeMux)
	}()
	// start register with connect retry
	const ttl = time.Second
	DefaultRegisterTTL = ttl
	cancel, err := NewEtcdRegistry(addr, "default").Register(ctx, ins, ttl)
	defer cancel()
	assert.NoError(t, err)
	// assert retry success
	assert.Eventually(t, func() bool {
		var rsp *clientv3.GetResponse
		rsp, err = client.Get(ctx, "/", clientv3.WithPrefix())
		assert.NoError(t, err)
		return len(rsp.Kvs) == 1
	}, ttl*10, time.Millisecond*100, "assert register success")
	// force close proxy server to verify keepalive retry
	forceCloseAllFunc()
	_ = proxyServer.Close()
	// assert etcd server ttl expire
	assert.Eventually(t, func() bool {
		var rsp *clientv3.GetResponse
		rsp, err = client.Get(ctx, "/", clientv3.WithPrefix())
		assert.NoError(t, err)
		return len(rsp.Kvs) == 0
	}, ttl*10, time.Millisecond*100, "assert etcd server ttl expire")
	// restart proxy server
	proxyServer, forceCloseAllFunc, err = simpleTCPProxy(addr[0], etcdAddrs[0])
	assert.NoError(t, err)
	// assert retry success
	assert.Eventually(t, func() bool {
		var rsp *clientv3.GetResponse
		rsp, err = client.Get(ctx, "/", clientv3.WithPrefix())
		assert.NoError(t, err)
		return len(rsp.Kvs) == 1
	}, ttl*10, time.Millisecond*100, "assert register success")
	_ = s
}

func startSingleNodeETCD(t *testing.T, name string, lPort int, aPort int) (*embed.Etcd, []string, func()) {
	cfg := embed.NewConfig()
	cfg.LogLevel = "warn"
	lpurl, _ := url.Parse("http://localhost:" + cast.ToString(lPort))
	apurl, _ := url.Parse("http://localhost:" + cast.ToString(lPort))
	lcurl, _ := url.Parse("http://localhost:" + cast.ToString(aPort))
	acurl, _ := url.Parse("http://localhost:" + cast.ToString(aPort))
	cfg.ListenPeerUrls = []url.URL{*lpurl}
	cfg.AdvertisePeerUrls = []url.URL{*apurl}
	cfg.ListenClientUrls = []url.URL{*lcurl}
	cfg.AdvertiseClientUrls = []url.URL{*acurl}
	cfg.Dir = "/tmp/" + name
	cfg.Name = "name"
	cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)
	s, err := embed.StartEtcd(cfg)
	assert.NoError(t, err)
	var addrs []string
	for _, v := range s.Clients {
		addrs = append(addrs, v.Addr().String())
	}
	return s, addrs, func() {
		s.Close()
		assert.NoError(t, os.RemoveAll(cfg.Dir))
	}
}

type testInstance struct {
	key   string
	value string
}

// GetKey implements Instance interface.
func (t *testInstance) GetKey() string {
	return t.key
}

// GetValue implements Instance interface.
func (t *testInstance) GetValue() string {
	return t.value
}

func simpleTCPProxy(listenAddr string, proxyAddr string) (net.Listener, func(), error) {
	forceCloseAllFunc := func() {}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, forceCloseAllFunc, err
	}
	var allConnections []net.Conn
	var mu sync.Mutex
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("accept err:%v", err)
				return
			}
			mu.Lock()
			allConnections = append(allConnections, conn)
			mu.Unlock()
			go func() {
				defer func() {
					_ = conn.Close()
				}()
				proxyConn, err := net.Dial("tcp", proxyAddr)
				if err != nil {
					log.Printf("proxy dial err:%v", err)
					return
				}
				go func() {
					_, _ = io.Copy(proxyConn, conn)
				}()
				_, _ = io.Copy(conn, proxyConn)
			}()
		}
	}()
	forceCloseAllFunc = func() {
		mu.Lock()
		defer mu.Unlock()
		for _, v := range allConnections {
			_ = v.Close()
		}
	}
	return ln, forceCloseAllFunc, nil
}
