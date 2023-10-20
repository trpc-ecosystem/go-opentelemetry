# Debug 

debug模块主要方便用户进行异常调试，目前支持打印invalid utf-8错误时具体的字段，后续支持打印染色数据，远端错误码映射配置等能力。
使用时通过设置环境变量OTEL_SDK_DEBUG，支持组合的方式打开多个debugger，支持的debugger如下：

## utf8 debugger

通过设置环境变量OTEL_SDK_DEBUG为utf8，框架在log/trace导出时遇到invalid utf-8错误时会自动打印相关信息。
