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

package prometheus

import (
	"context"
	"io"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/server"

	trpccodes "trpc.group/trpc-go/go-opentelemetry/oteltrpc/codes"
	"trpc.group/trpc-go/go-opentelemetry/sdk/metric"
)

const systemName = "trpc"

// StreamServerFilter is a server side stream filter  that instruments the server side stream rpc metrics
func StreamServerFilter() server.StreamFilter {
	return func(ss server.Stream, info *server.StreamServerInfo, handler server.StreamHandler) error {
		ctx := ss.Context()
		msg := trpc.Message(ctx)
		sr := metric.NewServerReporter(
			systemName,
			msg.CallerServiceName(),
			msg.CallerMethod(),
			msg.CalleeServiceName(),
			msg.CalleeMethod(),
			metric.WithServerMetrics(metric.DefaultServerMetrics),
			metric.WithServerRPCType(serverStreamType(info)))
		err := handler(&monitoredServerStream{Stream: ss, monitor: sr})
		code, _ := trpccodes.GetDefaultGetCodeFunc()(ctx, nil, err)
		sr.Handled(ctx, code)
		return err
	}
}

// StreamClientFilter is a client-side filter that instruments the client side stream rpc metrics
func StreamClientFilter() client.StreamFilter {
	return func(
		ctx context.Context,
		desc *client.ClientStreamDesc,
		streamer client.Streamer,
	) (client.ClientStream, error) {
		msg := trpc.Message(ctx)
		cr := metric.NewClientReporter(
			systemName,
			msg.CallerServiceName(),
			msg.CallerMethod(),
			msg.CalleeServiceName(),
			msg.CalleeMethod(),
			metric.WithClientMetrics(metric.DefaultClientMetrics),
			metric.WithClientRPCType(clientStreamType(desc)),
		)

		cs, err := streamer(ctx, desc)
		if err != nil { // err == io.EOF if close stream normally
			var code = "0"
			if err != io.EOF {
				code, _ = trpccodes.GetDefaultGetCodeFunc()(ctx, nil, err)
			}
			cr.Handled(ctx, code)
			return nil, err
		}

		return &monitoredClientStream{
			ClientStream: cs,
			monitor:      cr,
		}, nil
	}
}

func clientStreamType(desc *client.ClientStreamDesc) metric.RPCType {
	switch {
	case desc.ClientStreams && !desc.ServerStreams:
		return metric.ClientStream
	case !desc.ClientStreams && desc.ServerStreams:
		return metric.ServerStream
	case desc.ClientStreams && desc.ServerStreams:
		return metric.BidiStream
	default:
		return metric.Unary
	}
}

func serverStreamType(desc *server.StreamServerInfo) metric.RPCType {
	switch {
	case desc.IsClientStream && !desc.IsServerStream:
		return metric.ClientStream
	case !desc.IsClientStream && desc.IsServerStream:
		return metric.ServerStream
	case desc.IsClientStream && desc.IsServerStream:
		return metric.BidiStream
	default:
		return metric.Unary
	}
}

type monitoredClientStream struct {
	client.ClientStream
	monitor *metric.ClientReporter
}

// SendMsg rewrite client.ClientStream/SendMsg method
func (s *monitoredClientStream) SendMsg(m interface{}) error {
	timer := s.monitor.SendMessageTimer()
	err := s.ClientStream.SendMsg(m)
	timer.ObserveDuration()
	if err == nil {
		s.monitor.SentMessage()
	}
	return err
}

// RecvMsg rewrite client.ClientStream/RecvMsg method
func (s *monitoredClientStream) RecvMsg(m interface{}) error {
	timer := s.monitor.ReceiveMessageTimer()
	err := s.ClientStream.RecvMsg(m)
	timer.ObserveDuration()

	switch err {
	case nil:
		s.monitor.ReceivedMessage()
	case io.EOF:
		s.monitor.Handled(s.Context(), "0")
	default:
		code, _ := trpccodes.GetDefaultGetCodeFunc()(s.Context(), m, err)
		s.monitor.Handled(s.Context(), code)
	}
	return err
}

type monitoredServerStream struct {
	server.Stream
	monitor *metric.ServerReporter
}

// SendMsg rewrite server.Stream/SendMsg method
func (s *monitoredServerStream) SendMsg(m interface{}) error {
	if err := s.Stream.SendMsg(m); err != nil {
		return err
	}
	s.monitor.SentMessage()
	return nil
}

// RecvMsg rewrite server.Stream/RecvMsg method
func (s *monitoredServerStream) RecvMsg(m interface{}) error {
	if err := s.Stream.RecvMsg(m); err != nil {
		return err
	}
	s.monitor.ReceivedMessage()
	return nil
}
