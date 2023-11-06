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
	"strconv"

	"trpc.group/trpc-go/trpc-go/log"

	"trpc.group/trpc-go/go-opentelemetry/config"
	"trpc.group/trpc-go/go-opentelemetry/oteltrpc/logs"
)

// doFlowLog
func doFlowLog(ctx context.Context, flow *logs.FlowLog, options FilterOptions) {
	if options.TraceLogMode == config.LogModeDisable {
		// fast path
		return
	}
	// process rule
	for _, v := range options.TraceLogOption.Exclude {
		matchService := v.Service == "" ||
			v.Service == flow.Source.Name ||
			v.Service == flow.Target.Name
		matchMethod := v.Method == "" ||
			v.Method == flow.Source.Method ||
			v.Method == flow.Target.Method
		matchCode := v.Code == "" || v.Code == strconv.FormatInt(int64(flow.Status.Code), 10)
		if matchService && matchMethod && matchCode {
			return
		}
	}
	switch options.TraceLogMode {
	case config.LogModeMultiLine:
		log.DebugContextf(ctx, "%s", flow.MultilineString())
		return
	case config.LogModeOneLine, config.LogModeDefault:
	default:
	}
	log.DebugContextf(ctx, "%s", flow.OneLineString())
}
