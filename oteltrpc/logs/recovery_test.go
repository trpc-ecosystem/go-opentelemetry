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
	"testing"

	"github.com/stretchr/testify/assert"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func TestLogRecoveryFilter(t *testing.T) {
	ctx := trpc.BackgroundContext()
	req := &pb.HelloRequest{}
	t.Run("recovery", func(t *testing.T) {
		_, err := LogRecoveryFilter()(ctx, req, func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("panic")
		})
		trpcErr := &errs.Error{}
		assert.ErrorAs(t, err, &trpcErr)
		assert.EqualValues(t, errs.Code(err), errs.RetServerSystemErr)
		assert.EqualValues(t, errs.Msg(err), "panic")
	})
	t.Run("disable-recovery", func(t *testing.T) {
		opts := func(o *FilterOptions) {
			o.DisableRecovery = true
		}
		var recovered bool
		var pancErr interface{}
		func() {
			defer func() {
				if pancErr = recover(); pancErr != nil {
					recovered = true
				}
			}()
			_, _ = LogRecoveryFilter(opts)(ctx, req, func(ctx context.Context, req interface{}) (interface{}, error) {
				panic("panic")
			})
		}()
		assert.True(t, recovered)
		assert.EqualValues(t, "panic", pancErr)
	})
	t.Run("custom-DefaultRecoveryHandler", func(t *testing.T) {
		var recovered bool
		DefaultRecoveryHandler = func(ctx context.Context, err interface{}) error {
			recovered = true
			return errs.NewFrameError(errs.RetServerSystemErr, "custom")
		}
		_, err := LogRecoveryFilter()(ctx, req, func(ctx context.Context, req interface{}) (interface{}, error) {
			panic("panic")
		})
		assert.True(t, recovered)
		assert.EqualValues(t, errs.Code(err), errs.RetServerSystemErr)
		assert.EqualValues(t, errs.Msg(err), "custom")
	})
}
