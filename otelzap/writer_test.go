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

package otelzap

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

/*
benchmark writer result:
BenchmarkConvertToRecordV1-12                             667284 1707 ns/op  1096 B/op 31 allocs/op
BenchmarkConvertToRecordV2-12                            1397292 833.9 ns/op 568 B/op  16 allocs/op
BenchmarkReadSampleLevel-12                              4171437 286.2 ns/op 24 B/op   5 allocs/op
BenchmarkConvertToRecordV2AndReadSampleLevel-12          1000000 1111 ns/op  592 B/op  21 allocs/op
*/

var testData = []byte(`{"level":"debug","ts":1603700525.1244118, "sampled": "true", 
"caller":"maxprocs/maxprocs.go:47","msg":"maxprocs: Leaving GOMAXPROCS=12: CPU quota undefined"}`)

func TestConvertToRecordV2(t *testing.T) {
	raw := make(map[string]interface{})
	err := jsoniter.ConfigFastest.Unmarshal(testData, &raw)
	assert.NoError(t, err)
	v1 := convertToRecord(raw)
	iter := jsoniter.ConfigFastest.BorrowIterator(testData)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	v2, err := convertToRecordV2(iter)
	assert.NoError(t, err)
	assert.Equal(t, v1.String(), v2.String())
}

func TestReadSampledAndLevel(t *testing.T) {
	iter := jsoniter.ConfigFastest.BorrowIterator(testData)
	defer jsoniter.ConfigFastest.ReturnIterator(iter)
	sampled, level, err := readSampledAndLevel(iter, testData)
	assert.NoError(t, err)
	assert.Equal(t, true, sampled)
	assert.Equal(t, "debug", level)
}

func BenchmarkConvertToRecordV1(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		raw := make(map[string]interface{})
		err := jsoniter.ConfigFastest.Unmarshal(testData, &raw)
		assert.NoError(b, err)
		v1 := convertToRecord(raw)
		_ = v1
	}
}

func BenchmarkConvertToRecordV2(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		iter := jsoniter.ConfigFastest.BorrowIterator(testData)
		v2, err := convertToRecordV2(iter)
		assert.NoError(b, err)
		_ = v2
		jsoniter.ConfigFastest.ReturnIterator(iter)
	}
}

func BenchmarkReadSampleLevel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		iter := jsoniter.ConfigFastest.BorrowIterator(testData)
		_, _, err := readSampledAndLevel(iter, testData)
		assert.NoError(b, err)
		jsoniter.ConfigFastest.ReturnIterator(iter)
	}
}

func BenchmarkConvertToRecordV2AndReadSampleLevel(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		iter := jsoniter.ConfigFastest.BorrowIterator(testData)
		_, _, err := readSampledAndLevel(iter, testData)
		assert.NoError(b, err)
		v2, err := convertToRecordV2(iter)
		assert.NoError(b, err)
		_ = v2
		jsoniter.ConfigFastest.ReturnIterator(iter)
	}
}
