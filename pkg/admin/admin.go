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

// Package admin provides simple admin http server API.
package admin

import (
	"net/http"
	"net/http/pprof"

	"trpc.group/trpc-go/trpc-opentelemetry/pkg/zpage"
	"trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

// Server is admin server, wrap http.Server
type Server struct {
	srv *http.Server

	opts *Options
}

// NewServer creates a Server instance
func NewServer(opts ...Option) (*Server, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if err := o.validate(); err != nil {
		return nil, err
	}

	return &Server{
		srv:  &http.Server{Addr: o.addr, Handler: newRouter(o)},
		opts: o,
	}, nil
}

// Serve starts a http server and listen to serve
func (s *Server) Serve() error {
	return s.srv.ListenAndServe()
}

// HTTPServer returns http.Server in Server
func (s *Server) HTTPServer() *http.Server {
	return s.srv
}

func newRouter(o *Options) *http.ServeMux {
	mux := http.NewServeMux()
	if o.enablePrometheus {
		mux.Handle("/metrics", metric.LimitMetricsHandler())
	}
	if o.enablePprof {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
	// support hot switch
	if o.enableHotSwitch {
		mux.HandleFunc("/cmds/disabletrace", DisableTrace)
		mux.HandleFunc("/cmds/enabletrace", EnableTrace)
		mux.HandleFunc("/cmds/tracestatus", TraceStatus)
	}
	// add zPage handler
	if o.enableZPage {
		mux.HandleFunc("/debug/tracez", zpage.GetZPageHandlerFunc())
	}

	return mux
}
