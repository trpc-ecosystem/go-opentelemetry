# TRPC-Go metric Reporting

The following is the data reported by opentelemetry in trpc-go, which is convenient for users to query in Grafana and create panels using PromQL.

## Metrics

Since some attributes are common to many metrics, we extract some common indicators here to avoid lengthy documentation. The following common attribute names will be used to replace these duplicate attrs.
* common_attrs:
    * app: ${app} in trpc.yaml
    * server: ${server}
    * namespace: ${namespace}
    * env_name: ${env_name}
    * container_name: ${container_name}
* rpc_relation:
    * system_name: hard code: "trpc"
    * caller_service: msg.CallerServiceName()
    * caller_method: msg.CallerMethod()
    * callee_service: msg.CalleeServiceName()
    * callee_method: msg.CalleeMethod()

### callee metrics 

#### server_started_total
* type: counter
* desc: 
* attrs:
    * common_attrs
    * rpc_relation

#### server_handled_total
* type: counter
* desc: total number of requests received and processed by the server
* attrs: 
    * common_attrs
    * rpc_relation
    * code: ret code 
    * code_type: ret code type
        * success
        * exception
        * timeout
    * code_desc: ret code description

#### server_handled_seconds
* type: histogram
* desc: distribution of request processing time
* bucket: Supports custom buckets, default buckets are [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5] (counting the number of requests less than 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 0.5s, 1s, 5s, respectively)
* attrs:
    * common_attrs
    * rpc_relation

### caller metrics

#### client_started_total
* type: counter
* desc: total number of requests sent 
* attrs: 
    * common_attrs
    * rpc_relation

#### client_handled_total
* type: counter
* desc: total number of requests sent and completed (may receive a response or timeout) 
* attrs: 
    * common_attrs
    * rpc_relation
    * code:
    * code_type:
        * success
        * exception
        * timeout
    * code_desc:

#### client_handled_seconds
* type: histogram
* desc: distribution of request processing time
* bucket: Supports custom buckets, default buckets are [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 5] (counting the number of requests less than 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 0.5s, 1s, 5s, respectively)
* attrs:
    * common_attrs
    * rpc_relation

### System

#### cpu_cores
* type: gauge
* desc: cpu cores(the correct quota of the container can be obtained within the container_
* frequency: only reported when server started
* attrs:
    * common_attrs

#### server_panic_total
* type: counter
* desc: panic counter
* attrs:
    * common_attrs
    * system_name: hard code "trpc"

### user defined metrics
* Supports reporting of single-dimensional and multi-dimensional metric data for trpc-go.
* For the three data types of counter, gauge, and histogram, the metric name will be prefixed with trpc_counter_, trpc_gauge_, and trpc_histogram_, respectively.
* The uniqueness of the metric name for the same type of business indicator must be ensured, otherwise, it will trigger a panic.

## Trace

### Resource
all trace include following metric：
* tps.tenant.id: tenent id
* telemetry.sdk.language: hardcode "go"
* telemetry.sdk.name: hardcode "opentelemetry"
* service.name(optional): {app}_{server} in trpc_go.yaml 


#### common attributes
common attributes for caller and callee:
* net:
    * peer:
        * ip: caller ip 
        * port: caller port 
    * host:
        * ip: callee ip
        * port: callee port
        * name: callee hostname
* trpc:
    * caller_service: msg.CallerServiceName
    * caller_method: msg.CallerMethod
    * callee_service: msg.CalleeServiceName
    * callee_method: msg.CalleeMethod
    * namespace: msg.Namespace (Development | Production)
* tps:
    * dyeing: dyeing key, msg.DyeingKey
    * force.sample(optional): flag for force sampling 

### callee
* common attributes
* badage
* traceContext
* spanKind: SPAN_KIND_SERVER
* spanName: msg.ServerRPCName

### caller
* common attributes
* spanKind: SPAN_KIND_CLIENT
* spanName: msg.ClientRPCName
