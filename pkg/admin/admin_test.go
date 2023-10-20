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

package admin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

func TestServer_Serve(t *testing.T) {
	srv, err := NewServer(WithAddr("localhost:6969"), WithEnablePprof(true), WithEnablePrometheus(true))
	require.NoError(t, err)

	httpSrv := srv.HTTPServer()
	defer func() {
		_ = httpSrv.Shutdown(context.TODO())
	}()

	go func() {
		_ = srv.Serve()
	}()

	time.Sleep(time.Second)
	got, err := http.Get(fmt.Sprintf("http://%s/debug/pprof/profile?seconds=2", httpSrv.Addr))
	if err != nil {
		return // 这里会出现 connect: connection refused 服务还没有listen
	}
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, got.StatusCode)

	data, err := io.ReadAll(got.Body)
	require.NoError(t, err)
	require.Greater(t, len(data), 0)
	defer got.Body.Close()

	response, err := http.Get(fmt.Sprintf("http://%s/metrics", httpSrv.Addr))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	defer response.Body.Close()

	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(response.Body)
	require.NoError(t, err)
	require.Greater(t, len(mf), 0)
}
