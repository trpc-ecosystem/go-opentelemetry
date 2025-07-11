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

package logs

import (
	"context"
	"math/rand"
	"testing"

	"trpc.group/trpc-go/trpc-go/log"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func benchmarkSpanLog(i int, b *testing.B) {
	str := randStr(i)
	arg := []interface{}{"test"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spanLogf(context.Background(), log.LevelNil, str+" %s", arg)
	}
}

func BenchmarkSpanLogf10(b *testing.B) {
	benchmarkSpanLog(10, b)
}

func BenchmarkSpanLogf1000(b *testing.B) {
	benchmarkSpanLog(1000, b)
}

func BenchmarkSpanLogf100000(b *testing.B) {
	benchmarkSpanLog(100000, b)
}

func BenchmarkSpanLogf10000000(b *testing.B) {
	benchmarkSpanLog(10000000, b)
}
