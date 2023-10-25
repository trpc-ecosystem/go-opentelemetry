# TRPC-Go数据埋点

以下是 opentelemetry 在trpc-go中埋点的数据，方便用户在grafana中查询以及使用promql创建panel

## Metrics

由于一些attributes很多metrics都有，因此这里提取一些公共的指标，避免文档过长。后续用下面的公共attributes名代替这些重复的attrs
* common_attrs:
    * app: trpc.yaml 中 ${app}
    * server: ${server}
    * namespace: ${namespace}
    * env_name: ${env_name}
    * container_name: ${container_name}
* rpc_relation:
    * system_name: 系统名称，hard code: "trpc"
    * caller_service: 主调方服务名，取msg.CallerServiceName()
    * caller_method: 主调方方法名，取msg.CallerMethod()
    * callee_service: 被调方服务名，取msg.CalleeServiceName()
    * callee_method: 被调方方法名，取msg.CalleMethod()

### 被调

#### server_started_total
* type: counter
* desc: 服务器收到并开始处理的请求总数
* attrs:
    * common_attrs
    * rpc_relation


#### server_handled_total
* type: counter
* desc: 服务器处理完成的请求总数
* attrs: 
    * common_attrs
    * rpc_relation
    * code: 返回码
    * code_type: 返回码类型
        * success
        * exception
        * timeout
    * code_desc: 返回码描述

#### server_handled_seconds
* type: histogram
* desc: 请求处理时长的耗时分布
* bucket: 支持自定义bucket，默认bucket为[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5] (分别统计 小于5ms 10ms 25ms 50ms 100ms 250ms 0.5s 1s 5s的请求的个数)
* attrs:
    * common_attrs
    * rpc_relation

### 主调

#### client_started_total
* type: counter
* desc: 发出的请求总数
* attrs: 
    * common_attrs
    * rpc_relation

#### client_handled_total
* type: counter
* desc: 发出并完成的请求总数（可能收到响应或请求超时）
* attrs: 
    * common_attrs
    * rpc_relation
    * code: 返回码
    * code_type: 返回码类型
        * success
        * exception
        * timeout
    * code_desc: 返回码描述

#### client_handled_seconds
* type: histogram
* desc: 发出请求的处理时长的耗时分布
* bucket: 支持自定义bucket，默认bucket为[0.005, 0.01, 0.1, 0.5, 1, 5] (分别统计 小于5ms 10ms 100ms 0.5s 1s 5s的请求的个数)
* attrs:
    * common_attrs
    * rpc_relation

### System

#### cpu_cores
* type: gauge
* desc: CPU核数（在容器内部可以正确得到容器的配额）
* frequency: 仅服务启动时上报一次
* attrs:
    * common_attrs

#### server_panic_total
* type: counter
* desc: 服务panic的总数
* attrs:
    * common_attrs
    * system_name: hard code "trpc"

### 自定义metrics
* 支持trpc-go单维、多维指标数据的上报。
* 针对counter、gauge、histogram三种数据类型，会针对metric name分别加上trpc_counter_、trpc_gauge_、trpc_histogram_的前缀。
* 业务指标上报需要保证同类型指标的metric name的唯一性，否则会触发panic。

## Trace

### Resource
所有trace都会包含以下内容：

* tps.tenant.id: 租户id
* telemetry.sdk.language: SDK语言，hardcode "go"
* telemetry.sdk.name: SDK名, hardcode "opentelemetry"
* service.name(optional): service名，trpc.yaml中第一个service的name


#### common attributes
主调和被调都共有的一些attributes:
* net:
    * peer:
        * ip: 调用方ip
        * port: 调用方端口
    * host:
        * ip: 被调方ip
        * port: 被调方端口
        * name: 被调方hostname
* trpc:
    * caller_service: 主调方serviceName msg.CallerServiceName
    * caller_method: 主调方方法 msg.CallerMethod
    * callee_service: 被调方serviceName msg.CalleeServiceName
    * callee_method: 被调方方法 msg.CalleeMethod
    * namespace: 服务所在namespace msg.Namespace (Development | Production)
* tps:
    * dyeing: 染色key名 msg.DyeingKey
    * force.sample(optional): 强制采样标识



### 被调

* common attributes
* badage
* traceContext
* spanKind: SPAN_KIND_SERVER
* spanName: 被调方法名 msg.ServerRPCName

### 主调
* common attributes
* spanKind: SPAN_KIND_CLIENT
* spanName: 调用下游的方法名 msg.ClientRPCName
