#  Go SDK

## 快速上手

### 1. 使用trpctelemetry方式接入 (tRPC框架推荐使用此种方式)

1. 匿名引入tRPC拦截器

```go
import _ "trpc-system/go-opentelemetry/oteltrpc"
```

2. 修改tRPC框架配置文件

```yaml
server:
  filter:
    - opentelemetry         #在tRPC服务端处理过程，引入拦截器

client:
  filter:
    - opentelemetry         #在tRPC客户端处理过程，引入拦截器

plugins:
  log:                                            #日志配置
    default:                                      #默认日志的配置，可支持多输出
      - writer: file                              #本地文件日志
        level: error                               #本地文件滚动日志的级别
        writer_config:
          filename: ../log/trpc.log                 #本地文件滚动日志存放的路径
          max_size: 10                              #本地文件滚动日志的大小 单位 MB
          max_backups: 10                           #最大日志文件数
          max_age: 7                                #最大日志保留天数
          compress:  false                          #日志文件是否压缩

  telemetry: # 注意缩进层级关系
    opentelemetry:
      addr: your.own.cluster.addr:port   # 集群地址（检查环境域名是否可以正常解析）
      tenant_id: your-tenant-id              # 租户ID，default代表默认租户，（注意：切换为业务租户ID）
      sampler:
        fraction: 0.0001                     # 采样（0.0001代表每10000请求上报一次trace数据）
        sampler_server_addr: your.own.sampler.addr:port     # 染色元数据查询平台地址
        sync_interval: 1m                    # sync_interval为sampler定时更新采样元数据的频率，默认10s
        # 下面为设置特定被调采样率的例子，业务可按需设置.
          # special_fractions:                   # 可指定被调的采样率
          # - callee_service: service1          # 指定被调service
          #   default_fraction: 0.0003         # 默认采样率，该service1下未指定callee_method采样率的使用这个采样率
          #   callee_methods:                  # 可指定被调method的采样率
          #     - method: method1              # 指定具体的callee_methods采样率
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
        enabled: true # 远程Metrics开关，默认打开
        enable_register: true # 注册metrics到etcd，默认打开
        # metrics注册地址 metrics功能需要打开trpc_admin, 如果运行在123平台, 则自动开启
        registry_endpoints: ["your.own.registry.addr:port"] # etcd endpoint
        server_owner: # 服务负责人, 对于123平台会自动设置. 用于监控看板展示及告警. 多个以分号分隔.
        client_histogram_buckets: [.005, .01, .1, .5, 1, 5] # 可选配置，用户自定义客户端直方图buckets数组(要求递增，长度不超过10，类型为float64）
        server_histogram_buckets: [.005, .01, .025, .05, .1, .25, .5, 1, 5] # 可选配置，用户自定义server耗时直方图buckets数组(要求递增，长度不超过10，类型为float64）
        disable_rpc_method_mapping: false # 可选配置(default false). 设置为true后，上报metric时会对被调接口名进行原样上报
        # 非restful的http服务需要把disable_rpc_method_mapping设置为true，而restful服务则设置为false且需要使用metric.RegisterMethodMapping注册path与pattern映射关系，避免高基数问题
        # codes 可设置特定错误码的类型(错误码转义), 以便计算错误率/超时率/成功率和看板展示错误码描述.
        # 默认值: 0:成功success 21/101:超时timeout 其它:错误exception
        codes:
          - code: 21
            type: timeout
            description: server超时
          - code: 101
            type: timeout
            description: client超时
        # 下面为设置特定返回码的例子，业务可按需设置.
        #  - code: 100014
        #    type: success # type 为 success 表示 100014 这个返码(无论主被调)会被统计为成功。不区分主被调，如果担忧错误码冲突，可以设置 service 和 method 来限定生效的 service 和 method
        #    description: desc4 # 对这个返回码的具体描述
        #  - code: 100015
        #    type: exception # type 为 exception 表示 100015 是个异常的错误码。可在description里设置更详细的说明信息
        #    description: desc5
        #    service: # 不为空表示错误码特例仅匹配特定的(无论主被调) service, 为空表示所有 service.
        #    method: # 不为空表示错误码特例仅匹配特定的(无论主被调) method, 为空表示所有 method.
        prometheus_push: # 上报指标到prometheus gateway
          enabled: false # 启用上报，默认关闭， 参见https://prometheus.io/docs/practices/pushing/#should-i-be-using-the-pushgateway
          # 如需在程序退出后发送delete请求到push gateway，需在main()函数trpc.NewServer()之后添加defer metric.DeletePrometheusPush(),详见https://trpc-system/go-opentelemetry#4-metrcs插件配置
          url: "" # 上报地址, 如http://1.1.1.1:4318
          job: "reporter" # 名称，不能为空，默认为reporter
          interval: 60s # 上报间隔，默认60秒
          # use_basic_auth: false # 启用basic认证，默认关闭
          # username: "" # 认证账号
          # password: "" # 认证密码
          # grouping: # push分组，默认为空
          #   name1: value1
          # http_headers: # http头部，将会添加到push请求，默认为空
          #   X-HEADER1: v1
      logs:
        enabled: true # 远程日志开关，默认关闭
        addr: "" # your.own.collector.com:port，绝大多数情况这项都不填，除非你有自建接收opentelemetry log协议日志的collector需求
        tls:
          enabled: false # 开启tls
          insecure_skip_veriry: false # 校验服务器证书
        level: "info" # 日志级别，默认error
        enable_sampler: false # 是否启用采样器, 启用后只有当前请求命中采样时才会上报独立日志
        enable_sampler_error: false # 与 enable_sampler 配合使用，未采样请求，若日志级别高于 error， 也会触发上报
        # trace_log(follow log)模式,  枚举值可选:verbose/multiline/disable
        # verbose:以DEBUG级别打印flow log包括接口名、请求、响应、耗时. multiline: 多行美化打印. disable:不打印, 默认不打印
        trace_log_mode: "verbose"
        trace_log_option: # trace_log(follow log) 选项
        #  exclude: # 例如排除某些接口, 如文件上传下载等不关注包体的接口.
        #    - service: # 根据service排除, 为空表示所有 service.
        #      method: # 根据method排除, 为空表示所有 method.
        #      code: # 根据code排除, 为空表示所有code.
        disable_recovery: false # log filter默认会recovery panic并打印日志上报指标
        # rate_limit 为日志流控配置，开启此配置可减少重复日志的打印
        # 例如，tick = 1s，first = 10, thereafter = 10 表示1秒内同一条日志打印超过10条后，则每隔10条才再次打印这一条相同的日志
        # 此时在这1s的生效周期里，如果某个相同的日志本应打印100条，实际上传的条数为19
        # 这里定义"相同的日志"为内容和等级都完全相同的重复日志。注意这里不包括日志的fields。如果fields不同，但内容和等级相同也被视为"相同的日志"
        rate_limit:
           enable_rate_limit: false # 是否开启日志流控配置。若开启，请按照业务需求来配置tick，first和thereafter
           tick: 1s # tick是日志流控的生效周期（即从打印一条日志开始计时在tick时间后，无论触发限流与否，对同一条计数器会被置为零，重新开始计数)
           first: 100 # first是限流阈值，即相同的日志达到first条时触发限流
           thereafter: 3 # 触发限流后每thereafter条相同日志才会输出一条
      traces:
        disable_trace_body: false # trace对req和rsp的上报开关, true:关闭上报以提升性能, false:上报, 默认上报
        enable_deferred_sample: false # 是否开启延迟采样 在span结束后的导出采样, 额外上报出错的/高耗时的. 默认: disable
        deferred_sample_error: true # 采样出错的
        deferred_sample_slow_duration: 500ms # 采样耗时大于指定值的
        disable_parent_sampling: false  # 默认 false, 开启后将不使用上游的采样结果
        enable_zpage:  false # 默认false,开启后，本地开启processor导出span,在/debug/tracez进行查看
```

3. metrcs插件配置
默认开启注册到etcd，可关闭。
支持上报指标到prometheus gateway，开启后如需在程序退出后发送delete请求到push gateway，需在main()函数trpc.NewServer()之后添加defer metric.DeletePrometheusPush()，如：
```go
package main

import (
  "trpc.group/trpc-go/trpc-go"
  _ "trpc-system/go-opentelemetry/oteltrpc"
  "trpc-system/go-opentelemetry/sdk/metric"
)

func main() {
  s := trpc.NewServer()
  defer metric.DeletePrometheusPush()
  ...
}
```


### 2. 使用 opentelemetry sdk方式接入

如果业务使用的框架没有实现类似 trpc-go 的上报插件，也可直接使用 opentelemetry sdk 方式接入。 上报demo可参考 [example](./example)。
