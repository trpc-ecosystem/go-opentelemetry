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

// Package metric
package metric

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"trpc-system/go-opentelemetry/pkg/protocol/opentelemetry-ext/proto/operation"
	"trpc-system/go-opentelemetry/sdk/remote"
)

func TestWithConfigurator(t *testing.T) {
	os.Setenv("TPS_TRACE", "remote")
	addr := setupRemoteConfigServer(func() *operation.Operation {
		return &operation.Operation{
			Metric: &operation.Metric{
				Codes: []*operation.Code{
					{
						Type:        "success",
						Code:        2,
						Description: "desc2",
					},
				},
			},
		}
	})
	time.Sleep(time.Millisecond * 10) // wait sync
	configurator := remote.NewRemoteConfigurator(addr, 0, "", "", "")
	cfg := Config{
		Enabled:           true,
		RegistryEndpoints: []string{"127.0.0.1:2379"},
		Instance: Instance{
			TenantID: "default",
			Addr:     "127.0.0.1:9999",
		},
		Codes: []*Code{
			NewCode("1", CodeTypeSuccess, "desc1"),
		},
		Configurator: configurator,
	}
	time.Sleep(time.Millisecond * 10) // wait sync
	_ = SetupByConfig(cfg)
	defer func() {
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		prometheus.DefaultGatherer = prometheus.NewRegistry()
	}()
	t.Run("from local config", func(t *testing.T) {
		desc := DefaultCodeTypeFunc("1", "", "")
		assert.EqualValues(t, CodeTypeException, desc.Type)
		assert.Equal(t, "code!=0", desc.Description)
	})
	t.Run("from remote config", func(t *testing.T) {
		desc := DefaultCodeTypeFunc("2", "", "")
		assert.EqualValues(t, CodeTypeException, desc.Type)
		assert.Equal(t, "code!=0", desc.Description)
	})
}

func setupRemoteConfigServer(handler func() *operation.Operation) string {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return ""
	}
	s := grpc.NewServer()
	operation.RegisterOperationServiceServer(s, &testOperationServiceServer{
		getHandler: handler,
	})
	go func() {
		_ = s.Serve(ln)
	}()
	return ln.Addr().String()
}

type testOperationServiceServer struct {
	operation.UnimplementedOperationServiceServer
	getHandler func() *operation.Operation
}

func (ts *testOperationServiceServer) SetOperation(ctx context.Context, req *operation.SetOperationRequest) (
	rsp *operation.SetOperationResponse, err error) {
	return nil, nil
}

func (ts *testOperationServiceServer) GetOperation(ctx context.Context, req *operation.GetOperationRequest) (
	rsp *operation.GetOperationResponse, err error) {
	return &operation.GetOperationResponse{Operation: ts.getHandler()}, nil
}

// BenchmarkCodeTypeFunc_ByCode
// BenchmarkCodeTypeFunc_ByCode/hit-success
// BenchmarkCodeTypeFunc_ByCode/hit-success-12         	172989316	         6.801 ns/op
// BenchmarkCodeTypeFunc_ByCode/miss-exception
// BenchmarkCodeTypeFunc_ByCode/miss-exception-12      	90388371	        12.19 ns/op
func BenchmarkCodeTypeFunc_ByCode(b *testing.B) {
	codes := []*Code{
		NewCode("0", CodeTypeSuccess, "code=0"),
		NewCode("", CodeTypeSuccess, "code="),
		NewCode("101", CodeTypeTimeout, "client timeout"),
		NewCode("21", CodeTypeTimeout, "server timeout"),
	}
	codeMapping := make(map[string]*Code, len(codes))
	for _, v := range codes {
		codeMapping[v.Code] = v
	}
	var atomicCodeMapping atomic.Value
	atomicCodeMapping.Store(codeMapping)
	defaultCodeTypeFuncByCode := func(code string) *Code {
		if ret, ok := atomicCodeMapping.Load().(map[string]*Code)[code]; ok {
			return ret
		}
		return defaultCodeTypeFunc(code, "", "")
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.Run("hit-success", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			defaultCodeTypeFuncByCode("0")
		}
	})
	b.Run("miss-exception", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			defaultCodeTypeFuncByCode("1")
		}
	})
}

// BenchmarkCodeTypeFunc_ByCodeServiceMethod
// BenchmarkCodeTypeFunc_ByCodeServiceMethod/hit-success
// BenchmarkCodeTypeFunc_ByCodeServiceMethod/hit-success-12         	99706222	        11.44 ns/op
// BenchmarkCodeTypeFunc_ByCodeServiceMethod/hit-mapping-list
// BenchmarkCodeTypeFunc_ByCodeServiceMethod/hit-mapping-list-12    	43011308	        27.43 ns/op
// BenchmarkCodeTypeFunc_ByCodeServiceMethod/miss-mapping-list
// BenchmarkCodeTypeFunc_ByCodeServiceMethod/miss-mapping-list-12   	41413572	        27.94 ns/op
func BenchmarkCodeTypeFunc_ByCodeServiceMethod(b *testing.B) {
	codes := map[string][]*Code{
		"0":   {NewCode("0", CodeTypeSuccess, "code=0")},
		"":    {NewCode("", CodeTypeSuccess, "code=")},
		"101": {NewCode("101", CodeTypeTimeout, "client timeout")},
		"21":  {NewCode("21", CodeTypeTimeout, "server timeout")},
		"1": {
			{
				Type:        CodeTypeSuccess.String(),
				Description: "code=1",
				Service:     "test.serviceName1",
				Method:      "testMethodName1",
			},
			{
				Type:        CodeTypeSuccess.String(),
				Description: "code=1",
				Service:     "test.serviceName2",
				Method:      "",
			},
		},
	}
	var atomicCodeMapping atomic.Value
	atomicCodeMapping.Store(codes)
	defaultCodeTypeFuncByCode := func(code string, service string, method string) *Code {
		if v, ok := atomicCodeMapping.Load().(map[string][]*Code)[code]; ok {
			for _, vv := range v {
				if vv.Service == "" && vv.Method == "" {
					return vv
				}
				if vv.Service == service && vv.Method == "" {
					return vv
				}
				if vv.Service == "" && vv.Method == method {
					return vv
				}
				if vv.Service == service && vv.Method == method {
					return vv
				}
			}
		}
		return defaultCodeTypeFunc(code, service, method)
	}
	b.ReportAllocs()
	b.ResetTimer()
	b.Run("hit-success", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			defaultCodeTypeFuncByCode("0", "test.serviceName1", "testMethodName1")
		}
	})
	b.Run("hit-mapping-list", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			defaultCodeTypeFuncByCode("1", "test.serviceName2", "testMethodName1") // miss
		}
	})
	b.Run("miss-mapping-list", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			defaultCodeTypeFuncByCode("1", "test.serviceName2", "testMethodName1") // miss
		}
	})
}

type requestInfo struct {
	lastMethod string
	lastBody   []byte
	lastPath   string
	lastHeader http.Header
}

func newPushGatewayOK(lastReq *requestInfo, username, password string) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if username != "" {
				u, p, ok := r.BasicAuth()
				if !ok {
					http.Error(w, "auth required", http.StatusUnauthorized)
					return
				}
				if u != username {
					http.Error(w, "auth failed", http.StatusUnauthorized)
					return
				}
				if p != password {
					http.Error(w, "auth failed", http.StatusUnauthorized)
					return
				}
			}
			lastReq.lastMethod = r.Method
			var err error
			lastReq.lastBody, err = ioutil.ReadAll(r.Body)
			if err != nil {
				panic(err)
			}
			lastReq.lastPath = r.URL.EscapedPath()
			lastReq.lastHeader = r.Header
			w.Header().Set("Content-Type", `text/plain; charset=utf-8`)
			if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusAccepted)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)
}

func newPrometheusRegistry() *prometheus.Registry {
	metric1 := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "testname1",
		Help: "testhelp1",
	})
	metric2 := prometheus.NewGauge(prometheus.GaugeOpts{
		Name:        "testname2",
		Help:        "testhelp2",
		ConstLabels: prometheus.Labels{"foo": "bar", "dings": "bums"},
	})

	reg := prometheus.NewRegistry()
	reg.MustRegister(metric1)
	reg.MustRegister(metric2)

	return reg
}

func newPushGatewayErr() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "fake error", http.StatusInternalServerError)
		}),
	)
}

func TestPrometheusPush(t *testing.T) {
	var (
		lastReq  = &requestInfo{}
		job      = "test_job"
		username = "test_user"
		password = "test_pass"
	)
	// test all ok
	pgwOK := newPushGatewayOK(lastReq, username, password)
	defer pgwOK.Close()

	reg := newPrometheusRegistry()

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, expfmt.FmtProtoDelim)

	for _, mf := range mfs {
		if err = enc.Encode(mf); err != nil {
			t.Fatal(err)
		}
	}
	wantBody := buf.Bytes()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	err = SetupByConfig(Config{
		Enabled:         true,
		EnabledRegister: false,
		PrometheusPush: PrometheusPushConfig{
			Enabled:      true,
			URL:          pgwOK.URL,
			Job:          job,
			Interval:     0,
			UseBasicAuth: true,
			Username:     username,
			Password:     password,
			Grouping: map[string]string{
				"g1": "v1",
				"g2": "v2",
			},
			HTTPHeaders: map[string]string{
				"h1": "v1",
				"h2": "v2",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		prometheus.DefaultGatherer = prometheus.NewRegistry()
	}()
	if lastReq.lastMethod != http.MethodPut {
		t.Errorf("got method %q for Push, want %q", lastReq.lastMethod, http.MethodPut)
	}
	if !bytes.Equal(lastReq.lastBody, wantBody) {
		t.Errorf("got body %v, want %v", lastReq.lastBody, wantBody)
	}
	if lastReq.lastPath != "/metrics/job/"+job+"/g1/v1/g2/v2" && lastReq.lastPath != "/metrics/job/"+job+"/g2/v2/g1/v1" {
		t.Error("unexpected path:", lastReq.lastPath)
	}
	if lastReq.lastHeader.Get("h1") != "v1" || lastReq.lastHeader.Get("h2") != "v2" {
		t.Error("unexpected header:", lastReq.lastHeader)
	}

	// test 401 auth required
	reg = newPrometheusRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	if err := SetupByConfig(Config{
		Enabled:         true,
		EnabledRegister: false,
		PrometheusPush: PrometheusPushConfig{
			Enabled:      true,
			URL:          pgwOK.URL,
			Job:          job,
			Interval:     0,
			UseBasicAuth: false,
		},
	}); err == nil {
		t.Error("push to auth required Pushgateway succeeded")
	} else {
		if got, want := err.Error(), fmt.Sprintf(
			"unexpected status code 401 while pushing to %s/metrics/job/%s: auth required\n",
			pgwOK.URL, job,
		); got != want {
			t.Errorf("got error %q, want %q", got, want)
		}
	}

	// test 401 auth failed
	reg = newPrometheusRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	if err := SetupByConfig(Config{
		Enabled:         true,
		EnabledRegister: false,
		PrometheusPush: PrometheusPushConfig{
			Enabled:      true,
			URL:          pgwOK.URL,
			Job:          job,
			Interval:     0,
			UseBasicAuth: true,
			Username:     username + "a",
			Password:     password + "a",
		},
	}); err == nil {
		t.Error("push to auth required Pushgateway with wrong auth info succeeded")
	} else {
		if got, want := err.Error(), fmt.Sprintf(
			"unexpected status code 401 while pushing to %s/metrics/job/%s: auth failed\n",
			pgwOK.URL, job,
		); got != want {
			t.Errorf("got error %q, want %q", got, want)
		}
	}

	// test 500 fake error
	pgwErr := newPushGatewayErr()
	defer pgwErr.Close()
	reg = newPrometheusRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	if err := SetupByConfig(Config{
		Enabled:         true,
		EnabledRegister: false,
		PrometheusPush: PrometheusPushConfig{
			Enabled:  true,
			URL:      pgwErr.URL,
			Job:      job,
			Interval: 0,
		},
	}); err == nil {
		t.Error("push to broken Pushgateway succeeded")
	} else {
		if got, want := err.Error(), fmt.Sprintf(
			"unexpected status code 500 while pushing to %s/metrics/job/%s: fake error\n",
			pgwErr.URL, job,
		); got != want {
			t.Errorf("got error %q, want %q", got, want)
		}
	}
}
