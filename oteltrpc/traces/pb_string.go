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

package traces

import (
	"context"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ProtoMessageToJSONBytes .
func ProtoMessageToJSONBytes(message interface{}) []byte {
	out, err := jsoniter.ConfigFastest.Marshal(message)
	if err == nil {
		return out
	}
	// unexpected
	return []byte("")
}

// ProtoMessageToJSONString use json-iteration marshaller
func ProtoMessageToJSONString(message interface{}) string {
	out, err := jsoniter.ConfigFastest.MarshalToString(message)
	if err == nil {
		return out
	}
	// unexpected
	return ""
}

// ProtoMessageToJSONIndentBytes return pretty string bytes for multiple lines with indent
func ProtoMessageToJSONIndentBytes(message interface{}) []byte {
	out, err := jsoniter.ConfigFastest.MarshalIndent(message, "", "  ")
	if err == nil {
		return out
	}
	// unexpected
	return []byte("")
}

// ProtoMessageToJSONIndentString return pretty string for multiple lines
func ProtoMessageToJSONIndentString(message interface{}) string {
	out, err := jsoniter.ConfigFastest.MarshalIndent(message, "", "  ")
	if err == nil {
		return yoloString(out)
	}
	// unexpected
	return ""
}

// ProtoMessageToPBJSONString use pbjson serialization，convert uint64 to string
func ProtoMessageToPBJSONString(message interface{}) string {
	if p, ok := message.(proto.Message); ok {
		if out, err := protojson.Marshal(p); err == nil {
			return yoloString(out)
		}
	}
	// unexpected
	return ""
}

// ProtoMessageToCustomJSONString use json-iterator serialization, convert int64/uint64 to string
func ProtoMessageToCustomJSONString(message interface{}) string {
	out, err := Integer64AsStringConfig.MarshalToString(message)
	if err == nil {
		return out
	}
	// unexpected
	return ""
}

// ProtoMessageToCustomJSONStringWithContext use DefaultTraceEventMsgMarshaller
func ProtoMessageToCustomJSONStringWithContext(ctx context.Context, message interface{}) string {
	return DefaultTraceEventMsgMarshaler(message)
}

func yoloString(b []byte) string {
	return *((*string)(unsafe.Pointer(&b)))
}
