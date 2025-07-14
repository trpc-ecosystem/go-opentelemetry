//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

// SayHello say hello request
// trpc-cli -func /trpc.test.helloworld.Greeter/SayHello -target ip://127.0.0.1:8000 -body '{"msg":"hellotrpc"}'
// curl -X Post -d '{"msg":"hellopost"}' -H "Content-Type:application/json"
// http://127.0.0.1:8080/trpc.test.helloworld.Greeter/SayHello
// curl http://127.0.0.1:8080/trpc.test.helloworld.Greeter/SayHello?msg=helloget
// -H "trpc-trans-info: {\"tps-force-sample\":\"$(echo -n "demo-force-sample"|base64)\"}" -v
func (s *GreeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.DebugContextf(ctx, "SayHello recv req:%s", req)
	metrics.IncrCounter("SayHello请求量", 1)
	metrics.SetGauge("SayHello请求长度", float64(len(req.Msg)))
	md := trpc.Message(ctx)
	log.DebugContextf(ctx, "got meta %+v", md)
	if strings.HasPrefix(req.Msg, "errorCode") {
		code, _ := strconv.ParseInt(strings.TrimPrefix(req.Msg, "errorCode"), 10, 64)
		return nil, errs.New(int(code), req.Msg)
	}
	if strings.HasPrefix(req.Msg, "sleep") {
		duration, _ := time.ParseDuration(strings.TrimPrefix(req.Msg, "sleep"))
		time.Sleep(duration)
	}
	proxy := pb.NewGreeterClientProxy()
	hi, err := proxy.SayHi(ctx, req)
	if err != nil {
		log.ErrorContextf(ctx, "say hi fail:%v", err)
		return nil, err
	}

	return &pb.HelloReply{
		Msg: "Hello " + hi.Msg,
	}, nil
}

// SayHi say hi request
func (s *GreeterServerImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.DebugContextf(ctx, "SayHi recv req:%s", req)

	return &pb.HelloReply{
		Msg: "Hi" + req.Msg,
	}, nil
}
