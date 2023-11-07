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

//go:build !race
// +build !race

package prometheus

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"

	pb "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc/example/testproto"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

var (
	_ prometheus.Collector = metric.NewClientMetrics()
	_ prometheus.Collector = metric.NewServerMetrics()
)

var countListResponses = 20

func TestClientFilterSuite(t *testing.T) {
	suite.Run(t, new(clientFilterTestSuite))
}

func TestServerFilterSuite(t *testing.T) {
	suite.Run(t, new(serverFilterTestSuite))
}

type fakerServer struct {
	originTrpcConfigPath string

	svr            *server.Server
	clientProxy    pb.TestServiceClientProxy
	testServerImpl *testServer

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (f *fakerServer) init(t *testing.T) {
	f.originTrpcConfigPath = trpc.ServerConfigPath
	trpc.ServerConfigPath = initConfigFile(t)

	f.svr = trpc.NewServer(server.WithStreamFilter(StreamServerFilter()))
	f.clientProxy = pb.NewTestServiceClientProxy(client.WithStreamFilter(StreamClientFilter()))
	f.testServerImpl = &testServer{t: t}

	pb.RegisterTestServiceService(f.svr, f.testServerImpl)
	go func() {
		_ = f.svr.Serve()
	}()
}

type clientFilterTestSuite struct {
	suite.Suite

	*fakerServer
}

func (c *clientFilterTestSuite) SetupSuite() {
	metric.DefaultClientMetrics.Reset()

	ctx, cancel := context.WithCancel(context.TODO())
	c.fakerServer = &fakerServer{ctx: ctx, cancelFunc: cancel}
	c.fakerServer.init(c.T())
}

func (c *clientFilterTestSuite) SetupTest() {
	c.ctx, c.cancelFunc = context.WithTimeout(context.TODO(), 2*time.Second)

	// reset all metrics state
	metric.DefaultClientMetrics.Reset()
}

func (c *clientFilterTestSuite) TearDownSuite() {
	trpc.ServerConfigPath = c.originTrpcConfigPath
	_ = c.svr.Close(nil)
}

func (c *clientFilterTestSuite) TearDownTest() {
	c.cancelFunc()
}

func (c *clientFilterTestSuite) TestStartedStreamingIncrementsStarted() {
	counter := metric.DefaultClientMetrics.ClientStartedCounter().WithLabelValues(
		"trpc", "-", "-", "trpc.test.testproto.TestService", "PingList")

	_, err := c.clientProxy.PingList(c.ctx, &pb.PingRequest{})
	require.NoError(c.T(), err)
	requireValue(c.T(), 1, counter)

	_, err = c.clientProxy.PingList(c.ctx, &pb.PingRequest{ErrorCodeReturned: uint32(100)})
	require.NoErrorf(c.T(), err, "PingList must not fail immediately")
	requireValue(c.T(), 2, counter)
}

func (c *clientFilterTestSuite) TestStreamingIncrementsMetrics() {
	ss, err := c.clientProxy.PingList(c.ctx, &pb.PingRequest{})
	require.NoError(c.T(), err)

	var count int
	for {
		_, err := ss.Recv()
		if err == io.EOF {
			break
		}
		require.NoErrorf(c.T(), err, "reading pingList should not fail")
		count++
	}
	require.EqualValues(c.T(), countListResponses, count, "Number of received msg on the wire must match")

	labels := []string{"trpc", "client_stream", "-", "-", "trpc.test.testproto.TestService", "PingList"}
	defaultMetrics := metric.DefaultClientMetrics
	requireValue(c.T(), countListResponses, defaultMetrics.ClientStreamMsgReceived().WithLabelValues(labels...))
	requireValue(c.T(), 1, defaultMetrics.ClientStreamMsgSent().WithLabelValues(labels...))
}

type serverFilterTestSuite struct {
	suite.Suite

	*fakerServer
}

func (s *serverFilterTestSuite) SetupSuite() {
	metric.DefaultServerMetrics.Reset()

	s.fakerServer = new(fakerServer)
	s.fakerServer.init(s.T())
}

func (s *serverFilterTestSuite) SetupTest() {
	s.ctx, s.cancelFunc = context.WithTimeout(trpc.BackgroundContext(), 2*time.Second)

	// reset all metrics state
	metric.DefaultServerMetrics.Reset()
}

func (s *serverFilterTestSuite) TearDownSuite() {
	trpc.ServerConfigPath = s.originTrpcConfigPath
	_ = s.svr.Close(nil)
}

func (s *serverFilterTestSuite) TearDownTest() {
	s.cancelFunc()
}

func (s *serverFilterTestSuite) TestStartedStreamingIncrementsStarted() {
	counter := metric.DefaultServerMetrics.ServerStartedCounter().WithLabelValues(
		"trpc", "trpc.test.testproto.TestService", "-", "trpc.test.testproto.TestService", "PingList")

	_, err := s.clientProxy.PingList(s.ctx, &pb.PingRequest{})
	require.NoError(s.T(), err)
	requireValueWithRetry(s.ctx, s.T(), 1, counter)

	_, err = s.clientProxy.PingList(s.ctx, &pb.PingRequest{ErrorCodeReturned: uint32(10)})
	require.NoError(s.T(), err, "PingList must not fail immediately")
	requireValueWithRetry(s.ctx, s.T(), 2, counter)
}

func (s *serverFilterTestSuite) TestStreamingIncrementsMetrics() {
	ss, _ := s.clientProxy.PingList(s.ctx, &pb.PingRequest{}) // should return with code=OK
	// Do a read, just for kicks.
	count := 0
	for {
		_, err := ss.Recv()
		if err == io.EOF {
			break
		}
		require.NoError(s.T(), err, "reading pingList shouldn't fail")
		count++
	}
	require.EqualValues(s.T(), countListResponses, count, "Number of received msg on the wire must match")

	m := metric.DefaultServerMetrics
	requireValueWithRetry(s.ctx, s.T(), 1,
		m.ServerStartedCounter().WithLabelValues(
			systemName,
			"trpc.test.testproto.TestService",
			"-",
			"trpc.test.testproto.TestService",
			"PingList",
		))
	requireValueWithRetry(s.ctx, s.T(), countListResponses,
		m.ServerStreamMsgSent().WithLabelValues(
			systemName,
			"server_stream",
			"trpc.test.testproto.TestService",
			"-",
			"trpc.test.testproto.TestService",
			"PingList",
		))
	requireValueWithRetry(s.ctx, s.T(), 1,
		m.ServerStreamMsgReceived().WithLabelValues(
			systemName,
			"server_stream",
			"trpc.test.testproto.TestService",
			"-",
			"trpc.test.testproto.TestService",
			"PingList",
		))

	_, err := s.clientProxy.PingList(s.ctx, &pb.PingRequest{ErrorCodeReturned: 10})
	require.NoError(s.T(), err, "PingList must not fail immediately")

	requireValueWithRetry(s.ctx, s.T(), 2,
		m.ServerStartedCounter().WithLabelValues(
			systemName,
			"trpc.test.testproto.TestService",
			"-",
			"trpc.test.testproto.TestService",
			"PingList",
		))
}

type testServer struct {
	t *testing.T
}

func (t *testServer) PingList(request *pb.PingRequest, stream pb.TestService_PingListServer) error {
	if request.GetErrorCodeReturned() != 0 {
		return errs.New(10000, "test error")
	}

	for i := 0; i < countListResponses; i++ {
		if err := stream.Send(&pb.PingResponse{Value: request.Value, Counter: int32(i)}); err != nil {
			return err
		}
	}
	return nil
}

func requireValue(t *testing.T, expect int, c prometheus.Collector) {
	t.Helper()
	v := int(testutil.ToFloat64(c))
	if v == expect {
		return
	}

	metricFullName := reflect.ValueOf(*c.(prometheus.Metric).Desc()).FieldByName("fqName").String()
	t.Errorf("expected %d %s value; got %d; ", expect, metricFullName, v)
	t.Fail()
}

func requireValueWithRetry(ctx context.Context, t *testing.T, expect int, c prometheus.Collector) {
	t.Helper()
	for {
		v := int(testutil.ToFloat64(c))
		if v == expect {
			return
		}

		select {
		case <-ctx.Done():
			metricFullName := reflect.ValueOf(*c.(prometheus.Metric).Desc()).FieldByName("fqName").String()
			t.Errorf("timeout while expecting %d %s value; got %d; ", expect, metricFullName, v)
			t.Fail()
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func initConfigFile(t *testing.T) string {
	port, err := getFreePort()
	require.NoError(t, err)

	f, err := os.CreateTemp(t.TempDir(), "trpc_go.yaml")
	require.NoError(t, err)

	_, err = f.WriteString(fmt.Sprintf(`global:                             #全局配置
  namespace: Development            #环境类型，分正式Production和非正式Development两种类型
  env_name: test                    #环境名称，非正式环境下多环境的名称

server:                                            #服务端配置
  app: test                                        #业务的应用名
  server: testproto                                #进程服务名
  bin_path: /usr/local/trpc/bin/                   #二进制可执行文件和框架配置文件所在路径
  conf_path: /usr/local/trpc/conf/                 #业务配置文件所在路径
  data_path: /usr/local/trpc/data/                 #业务数据文件所在路径
  service:                                         #业务服务提供的service，可以有多个
    - name: trpc.test.testproto.TestService        #service的名字服务路由名称
      ip: 127.0.0.1                                #服务监听ip地址
      port: %d                                     #服务监听端口
      network: tcp                                 #网络监听类型 tcp udp
      protocol: trpc                               #应用层协议 trpc http
      timeout: 1000                                #请求最长处理时间 单位 毫秒

client:                                            #客户端调用的后端配置
  timeout: 1000                                    #针对所有后端的请求最长处理时间
  namespace: Development                           #针对所有后端的环境
  service:                                         #针对单个后端的配置
    - callee: trpc.test.testproto.TestService      #后端服务协议文件的service name, 如果callee和下面的name一样，那只需要配置一个即可
      name: trpc.test.testproto.Greeter        	   #后端服务名字路由的service name，有注册到名字服务的话，下面target可以不用配置
      target: ip://127.0.0.1:%d                    #后端服务地址
      network: tcp                                 #后端服务的网络类型 tcp udp
      protocol: trpc                               #应用层协议 trpc http
      timeout: 800                                 #请求最长处理时间
      serialization: 0                             #序列化方式 0-pb 1-jce 2-json 3-flatbuffer，默认不要配置

plugins:                                           #插件配置
  log:                                             #日志配置
    default:                                       #默认日志的配置，可支持多输出
      - writer: console                            #控制台标准输出 默认
        level: debug                               #标准输出日志的级别`, port, port))
	require.NoError(t, err)
	return f.Name()
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
