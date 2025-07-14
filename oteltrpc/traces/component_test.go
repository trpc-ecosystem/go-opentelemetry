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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
)

func TestDatabaseAttribute(t *testing.T) {
	msg := trpc.Message(context.TODO())

	var commonMeta = codec.CommonMeta{}
	commonMeta[semconv.DBSystemMongoDB.Key] = semconv.DBSystemMongoDB.Value.AsString()
	commonMeta[semconv.DBStatementKey] = "InsertOne"
	commonMeta[semconv.DBNameKey] = "trpc.mongo"
	commonMeta[semconv.DBMongoDBCollectionKey] = "schoool"
	commonMeta[semconv.DBSQLTableKey] = "table"

	msg.WithCommonMeta(commonMeta)

	attrs := databaseAttribute(msg)
	assert.Equal(t, len(attrs), 5)
}
