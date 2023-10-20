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

package log

import (
	"context"

	logsproto "go.opentelemetry.io/proto/otlp/logs/v1"
)

// Exporter exports telemetry log data from the collector to a destination.
type Exporter interface {
	ExportLogs(context.Context, []*logsproto.ResourceLogs) error
	Shutdown(ctx context.Context) error
}
