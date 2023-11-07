English | [中文](README.zh_CN.md)

#  Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/trpc-ecosystem/go-opentelemetry.svg)](https://pkg.go.dev/github.com/trpc-ecosystem/go-opentelemetry)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-opentelemetry)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-opentelemetry)
[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://github.com/trpc-ecosystem/go-opentelemetry/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release/trpc-ecosystem/go-opentelemetry.svg?style=flat-square)](https://github.com/trpc-ecosystem/go-opentelemetry/releases)
[![Tests](https://github.com/trpc-ecosystem/go-opentelemetry/actions/workflows/prc.yml/badge.svg)](https://github.com/trpc-ecosystem/go-opentelemetry/actions/workflows/prc.yml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-opentelemetry/branch/main/graph/badge.svg)](https://app.codecov.io/gh/trpc-ecosystem/go-opentelemetry/tree/main)

## Quickstart

### 1. Use trpctelemetry (recommended for tRPC)

1. import tRPC filter

```go
import _ "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc"
```

2. modify tRPC config

```yaml
server:
  filter:
    - opentelemetry         # server filter

client:
  filter:
    - opentelemetry         # client filter

plugins:
  log:                                            # log config
    default:                                      # default log config, support multiple log configs
      - writer: file                              # local file
        level: error                              # log level
        writer_config:
          filename: ../log/trpc.log                 # path for local log file
          max_size: 10                              # size for local log file: MB
          max_backups: 10                           # max log files
          max_age: 7                                # max days for log files 
          compress:  false                          # whether to compress log files

  telemetry:
    opentelemetry:
      addr: your.own.cluster.addr:port   # opentelemetry cluster address
      tenant_id: your-tenant-id              # tenant ID
      sampler:
        fraction: 0.0001                     # sampler fraction 
        sampler_server_addr: your.own.sampler.addr:port
        sync_interval: 1m                    # sync_interval default 10s
        # you can also set special fractions for different
          # special_fractions:
          # - callee_service: service1         # special callee service
          #   default_fraction: 0.0003         # default fraction for service1
          #   callee_methods:                  # special callee methods 
          #     - method: method1
          #       fraction: 0.004
          #     - method: method2
          #       fraction: 0.005
          # - callee_service: service2
          #   default_fraction: 0.0001
          #   callee_methods:
          #     - method: method1
          #       fraction: 0.06
          #     - method: method2
          #       fraction: 0.07
      metrics:
        enabled: true # default true
        enable_register: true # register metrics endpoint to etcd, default true
        registry_endpoints: ["your.own.registry.addr:port"]
        server_owner: # server owners separated by ;.
        client_histogram_buckets: [.005, .01, .1, .5, 1, 5] # optional config for client histogram buckets(Requires incrementing values, with a maximum length of 10 elements, and the data type should be float64.）
        server_histogram_buckets: [.005, .01, .025, .05, .1, .25, .5, 1, 5] # optional config for server histogram buckets(Requires incrementing values, with a maximum length of 10 elements, and the data type should be float64.）
        disable_rpc_method_mapping: false # Optional configuration (default false). When set to true, the original interface name will be reported as-is when reporting metrics.
        # For non-RESTful HTTP services, disable_rpc_method_mapping should be set to true, while for RESTful services, it should be set to false, and metric.RegisterMethodMapping should be used to register the path and pattern mapping relationship to avoid high cardinality issues.
        # Codes allow setting specific error code types (error code translation) for calculating error rate/timeout rate/success rate and displaying error code descriptions on dashboards. 
        codes:
          - code: 21
            type: timeout
            description: server timeout
          - code: 101
            type: timeout
            description: client timeout
        #  - code: 100014
        #    type: success # type set to 'success' indicates that the return code 100014 (regardless of whether it's a caller or callee) will be counted as successful. It does not differentiate between caller and callee. If you are concerned about error code conflicts, you can set 'service' and 'method' to restrict the effective service and method.
        #    description: desc4 # description for code 100014
        #  - code: 100015
        #    type: exception # type set to 'exception' indicates that the return code 100015 is an exceptional error code. You can set more detailed description information in the 'description' field.
        #    description: desc5 # description for code 100015
        #    service: # If not empty, it indicates that the error code exception only matches a specific service (regardless of whether it's a caller or callee). If empty, it applies to all services.
        #    method: # If not empty, it indicates that the error code exception only matches a specific method (regardless of whether it's a caller or callee). If empty, it applies to all methods.
        prometheus_push: # report to prometheus gateway
          enabled: false # default false， refer to  https://prometheus.io/docs/practices/pushing/#should-i-be-using-the-pushgateway
          # If you need to send a delete request to the push gateway after the program exits, add 'defer metric.DeletePrometheusPush()' after 'trpc.NewServer()' in the main() function. For more details, see https://trpc.group/trpc-go/go-opentelemetry#4-metrcs-plugin-configuration.
          url: "" # e.g., http://1.1.1.1:4318
          job: "reporter" # can't be empty, default: "reporter"
          interval: 60s # default 60 seconds
          # use_basic_auth: false # enable basic auth
          # username: ""
          # password: ""
          # grouping:
          #   name1: value1
          # http_headers:
          #   X-HEADER1: v1
      logs:
        enabled: true # remote log, default false 
        addr: "" # your.own.collector.com:port，
        tls:
          enabled: false
          insecure_skip_veriry: false
        level: "info" # default error
        enable_sampler: false
        enable_sampler_error: false # Used in conjunction with enable_sampler, for unsampled requests, if the log level is higher than error, reporting will also be triggered.
        # trace_log (follow log) mode, enumeration options: verbose/multiline/disable 
        # verbose: print flow log including interface name, request, response, and duration at DEBUG level. multiline: beautify print in multiple lines. disable: do not print, default is not printed.
        trace_log_mode: "verbose"
        trace_log_option:
        # exclude: # For example, exclude certain services, such as file upload and download services that do not focus on the packet body. 
        # - service: # Exclude based on service, empty means all services. 
        #   method: # Exclude based on method, empty means all methods.
        #   code: # Exclude based on code, empty means all codes.
        disable_recovery: false # By default, the log filter will recover from panics, print logs, and report metrics.
        # rate_limit is a log flow control configuration, enabling this configuration can reduce the printing of duplicate logs
        # For example, tick = 1s, first = 10, thereafter = 10 means that if the same log is printed more than 10 times within 1 second, then the same log will be printed again every 10 logs
        # At this time, in the 1s effective period, if a same log should be printed 100 times, the actual number of uploaded logs is 19
        # Here, a 'same log' is defined as a duplicate log with the same content and level. Note that this does not include the fields of the log. If the fields are different but the content and level are the same, they are also considered as the 'same log'.
        rate_limit:
           enable_rate_limit: false # Whether to enable log flow control configuration. If enabled, please configure tick, first, and thereafter according to business requirements.
           tick: 1s # tick is the effective period of log flow control (that is, starting from the printing of a log, regardless of whether flow control is triggered or not, the counter for the same log will be reset to zero and counting will restart after the tick time)
           first: 100 # first is the flow control threshold, that is, when the same log reaches the first number of occurrences, flow control is triggered
           thereafter: 3 # After flow control is triggered, every thereafter occurrences of the same log will output one log
      traces:
        disable_trace_body: false # Trace reporting switch for req and rsp, true: disable reporting to improve performance, false: report, report by default
        enable_deferred_sample: false # Whether to enable deferred sampling after the span ends, additionally reporting errors/high latency. Default: disable
        deferred_sample_error: true # Sample errors
        deferred_sample_slow_duration: 500ms # Sample durations greater than the specified value
        disable_parent_sampling: false  # Default false, when enabled, the upstream sampling result will not be used
        enable_zpage:  false # Default false, when enabled, the processor exports span locally and can be viewed at /debug/tracez
```

3. Metrics plugin setup
default registered to etcd cluster, can be turned off.
support prometheus gateway, require program sending delete request to push gateway before exit, add defer metric.DeletePrometheusPush() in main function, e.g.,
```go
package main

import (
  "trpc.group/trpc-go/trpc-go"
  _ "trpc.group/trpc-go/trpc-opentelemetry/oteltrpc"
  "trpc.group/trpc-go/trpc-opentelemetry/sdk/metric"
)

func main() {
  s := trpc.NewServer()
  defer metric.DeletePrometheusPush()
  ...
}
```


### 2.Use opentelemetry sdk

If the framework used by the business does not implement a reporting plugin similar to trpc-go, you can also directly integrate with the OpenTelemetry SDK. For a reporting demo, please refer to the following: [example](./example)。
