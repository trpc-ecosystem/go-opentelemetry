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

package metric

// RPCType is the type of RPC.
type RPCType string

// RPCType enum
const (
	Unary        RPCType = "unary"         // Unary RPC
	ClientStream RPCType = "client_stream" // Client-stream RPC
	ServerStream RPCType = "server_stream" // Server-stream RPC
	BidiStream   RPCType = "bidi_stream"   // Bidi-stream RPC
)
