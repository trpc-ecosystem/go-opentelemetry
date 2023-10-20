### 自定义exporter配置
如果业务的日志量比较大，天机阁默认的参数可能不适用，用户可以根据实际情况自行配置
```yaml
plugins:
  telemetry: # 注意缩进层级关系
    opentelemetry:
      addr: your.own.sever.ip:ip  # 集群地址（检查环境域名是否可以正常解析）
      tenant_id: your-tenant-id              # 租户ID，default代表默认租户，（注意：切换为业务租户ID）
      logs:
        enabled: true
        export_option: # 可选配置，默认配置可以满足绝大部分场景
          queue_size: 2048  # 本地内存中可以buffer的日志条数
          batch_size: 512  # 每次批量发送的日志条数，buffer中日志的条数超过这个配置就会触发日志发送行为
          batch_timeout: 5s  # 每隔5s触发一次日志发送行为
          max_batch_packet_size: 2097152  # buffer中日志总字节数超过2097152字节，会触发一次日志发送行为
```

#### 怎么看我是不是需要修改默认exporter配置？
* 在grafana执行 `sum(rate(opentelemetry_sdk_batch_process_counter{status=~"dropped|failed", telemetry="logs"}[5m])) by (server, status)`，查看是否有上报失败
* 如果某server有log被drop，说明queue_size小了，日志生产速率大于flush速率：
  * 可以调大queue_size。假如你的drop量是2000/s，那么你应该把queue_size调整到 >=2048+2000
  * 修改代码启动多通道上报
  * ```go
    import "trpc-system/go-opentelemetry/exporter/asyncexporter"
    
    func main() {
        asyncexporter.Concurrency = 3
    }
    ```
* 如果某server有log上报failed，说明到collector网络请求出问题，可能是collector请求压力比较大，可以尝试：
  * 调大batch_timeout，避免频繁上报
  * 调大batch_size，减少请求次数
  * 调大max_batch_packet_size，如果每条日志本身就很大，可以避免频繁发送请求到collector
  * 上述3个配置调整带来的风险就是，日志在flush到collector前在内存中暂存时间更长，如果遇到进程被kill容易丢失这部分数据
  * 数值不是越大越好，调整完后观察是否还有失败
