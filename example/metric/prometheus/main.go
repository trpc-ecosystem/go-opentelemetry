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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"

	"trpc-ecosystem/go-opentelemetry/sdk/metric"
)

const addr = ":8999"

func main() {
	instance := &metric.Instance{
		Addr:     "127.0.0.1" + addr,
		Metadata: map[string]string{},
	}
	err := metric.Setup(metric.WithRegistryEndpoints([]string{"your.own.registry.ip:Port"}),
		metric.WithInstance(instance),
		metric.WithTTL(time.Second*60))
	if err != nil {
		log.Fatal(err)
	}

	// add custom processor
	// set user-defined processor
	metric.SetProcessor(func(metrics []*dto.MetricFamily) []*dto.MetricFamily {
		var processedMetrics []*dto.MetricFamily
		for _, v := range metrics {
			if strings.HasPrefix(*v.Name, "client") { // filter metrics with the prefix client
				continue
			}
			processedMetrics = append(processedMetrics, v)
		}
		return processedMetrics
	})

	flag.Parse()
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/helloworld", handleHelloWorld)
	log.Fatal(http.ListenAndServe(addr, nil))
}

var httpCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: "",
		Name:      "http_request_handled_total",
		Help:      "http request handled total",
	},
	[]string{"status"},
)

func handleHelloWorld(w http.ResponseWriter, req *http.Request) {
	statusCode := http.StatusOK
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte("helloworld"))
	httpCounter.WithLabelValues(fmt.Sprint(statusCode)).Inc()
}
