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

package traces

import (
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

var (
	testData = &pb.HelloReply{
		Msg: "hello world!你好世界!",
	}
)

func TestProtoMessageToJSONBytes(t *testing.T) {
	t.Logf("%s", ProtoMessageToJSONBytes(testData))
}

func TestProtoMessageToJSONString(t *testing.T) {
	t.Logf("%s", ProtoMessageToJSONString(testData))
}

func TestProtoMessageToJSONIndentBytes(t *testing.T) {
	t.Logf("%s", ProtoMessageToJSONIndentBytes(testData))
}

func TestProtoMessageToJSONIndentString(t *testing.T) {
	t.Logf("%s", ProtoMessageToJSONIndentString(testData))
}

func TestProtoMessageToCustomJSONString(t *testing.T) {
	type args struct {
		message interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple pb",
			args: args{
				message: testData,
			},
			want: `{"msg":"hello world!你好世界!"}`,
		},
		{
			name: "int64 map",
			args: args{
				message: map[int64]int64{1: 1},
			},
			want: `{"1":"1"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ProtoMessageToCustomJSONString(tt.args.message); got != tt.want {
				t.Errorf("ProtoMessageToCustomJSONString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkProtoMessageToJSONBytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%v", ProtoMessageToJSONBytes(testData))
	}
}

func BenchmarkProtoMessageToJSONString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		//nolint: gosimple
		_ = fmt.Sprintf("%s", ProtoMessageToJSONString(testData))
	}
}

func BenchmarkProtoMessageToPBJSONString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		//nolint: gosimple
		_ = fmt.Sprintf("%s", ProtoMessageToPBJSONString(testData))
	}
}

func BenchmarkProtoMessageToCustomJSONString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		//nolint: gosimple
		_ = fmt.Sprintf("%s", ProtoMessageToCustomJSONString(testData))
	}
}

func BenchmarkProtoMessageToJSONIndentBytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = fmt.Sprintf("%v", ProtoMessageToJSONIndentBytes(testData))
	}
}

func BenchmarkProtoMessageToJSONIndentString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		//nolint: gosimple
		_ = fmt.Sprintf("%s", ProtoMessageToJSONIndentString(testData))
	}
}

func BenchmarkProtoMarshalTextString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		proto.MarshalTextString(testData)
	}
}

func BenchmarkProtoMarshalCompactTextString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = testData.String()
	}
}

func BenchmarkProtoMarshalPb(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(testData)
	}
}

// BenchmarkProtoSize-12    	 7018797	       165 ns/op	      16 B/op	       1 allocs/op
func BenchmarkProtoSize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = proto.Size(testData)
	}
}
