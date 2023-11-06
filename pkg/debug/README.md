# Debug

The debug module mainly facilitates users to debug exceptions. It currently supports printing specific fields for invalid utf-8 errors. It will later support printing of dyeing data, remote error code mapping configuration and other capabilities.
When using it, you can set the environment variable OTEL_SDK_DEBUG to support opening multiple debuggers in combination. The supported debuggers are as follows:

## utf8 debugger

By setting the environment variable OTEL_SDK_DEBUG to utf8, the framework will automatically print relevant information when encountering an invalid utf-8 error during log/trace export.