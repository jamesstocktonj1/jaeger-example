# Microservices Tracing Example

This is an example project of how to use [Jaeger](https://www.jaegertracing.io) tracing with the [Echo](https://echo.labstack.com) framework. Tracing is used to provide observability within distributed systems. In this example, we have a proxy which routes requests either to the account service or the booking service. When a request is sent to the booking service it calls a login endpoint in the account service. It then sends the booking request to RabbitMQ. The batch service then reads from the message queue and sends a request to the account service. This can all be traced in Jaeger.

## Architecture
![Jaeger Architecture](https://github.com/jamesstocktonj1/jaeger-example/blob/main/docs/jaeger_architecture.png)

## Trace
![Jaeger Architecture](https://github.com/jamesstocktonj1/jaeger-example/blob/main/docs/jaeger_trace.png)