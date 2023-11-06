### Custom Exporter Configuration
If the volume of logs in your business is large, the default parameters  may not be suitable, and users can configure them according to the actual situation.
```yaml
plugins:
  telemetry: # Note the indentation level relationship
    opentelemetry:
      addr: your.own.server.ip:port  # Cluster address (check if the environment domain can be resolved normally)
      tenant_id: your-tenant-id              # Tenant ID, default represents the default tenant (Note: Switch to the business tenant ID)
      logs:
        enabled: true
        export_option: # Optional configuration, default configuration can meet most scenarios
          queue_size: 2048  # The number of logs that can be buffered in local memory
          batch_size: 512  # The number of logs sent in batches each time, the log sending behavior will be triggered when the number of logs in the buffer exceeds this configuration
          batch_timeout: 5s  # Trigger log sending behavior every 5s
          max_batch_packet_size: 2097152  # When the total number of bytes of logs in the buffer exceeds 2097152 bytes, a log sending behavior will be triggered
```

#### How do I know if I need to modify the default exporter configuration?
* Execute sum(rate(opentelemetry_sdk_batch_process_counter{status=~"dropped|failed", telemetry="logs"}[5m])) by (server, status) to check if there are any reporting failures.
* If a server has logs dropped, it means the queue_size is too small, and the log production rate is greater than the flush rate:You can increase the queue_size. If your drop rate is 2000/s, then you should adjust the queue_size to >=2048+2000
  * You can increase the queue_size. If your drop rate is 2000/s, then you should adjust the queue_size to >=2048+2000
  * Modify the code to start multi-channel reporting
  * ```go
    import "trpc.group/trpc-go/go-opentelemetry/exporter/asyncexporter"
    
    func main() {
        asyncexporter.Concurrency = 3
    }
    ```
* If a server has logs reported as failed, it means there is a problem with the network request to the collector, which may be due to the high request pressure on the collector, you can try:
  * Increase the batch_timeout to avoid frequent reports
  * Increase batch_size to reduce the number of requests
  * Increase max_batch_packet_size, if each log is large, it can avoid frequent requests to the collector
  * The risk of adjusting the above three configurations is that logs will be stored in memory for a longer time before being flushed to the collector, and if the process is killed, this part of the data may be lost
  * The values are not the bigger, the better, adjust and observe if there are still failures
