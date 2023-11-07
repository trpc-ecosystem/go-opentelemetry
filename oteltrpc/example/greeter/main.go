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

package main

import (
	"log"

	"trpc.group/trpc-go/trpc-go"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	_ "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc" // trpctelemetry
)

// GreeterServerImpl greeter service implement
type GreeterServerImpl struct{}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	s := trpc.NewServer()
	pb.RegisterGreeterService(s, &GreeterServerImpl{})

	log.Fatal(s.Serve())
}
