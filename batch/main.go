package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/opentracing/opentracing-go"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

type QueueEntry struct {
	SpanID  string `json:"span_id"`
	Message string `json:"message"`
}

var ch *amqp.Channel
var chName string
var tracer opentracing.Tracer
var closer io.Closer

func main() {
	ch, err := initRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	err = initTracing()
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()

	msg, err := ch.Consume(
		chName, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatal(err)
	}

	for d := range msg {
		err := messageQueueHandler(d)
		if err != nil {
			log.Println(err)
		}
	}
}

func initRabbitMQ() (*amqp.Channel, error) {
	addr := os.Getenv("AMQP_ADDR")
	conn, err := amqp.Dial(fmt.Sprintf("amqp://user:password@%s:5672/", addr))
	if err != nil {
		return nil, err
	}

	ch, err = conn.Channel()
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"hello", // name
		false,   // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, err
	}
	chName = q.Name

	return ch, nil
}

func initTracing() error {
	// Add Opentracing instrumentation
	defcfg := config.Configuration{
		ServiceName: "echo-tracer",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
		},
	}
	cfg, err := defcfg.FromEnv()
	if err != nil {
		return err
	}
	tracer, closer, err = cfg.NewTracer()
	if err != nil {
		return err
	}

	opentracing.SetGlobalTracer(tracer)
	return nil
}

func messageQueueHandler(d amqp.Delivery) error {
	msg := QueueEntry{}
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		return err
	}

	ctx, err := getSpanContext(string(msg.SpanID))
	if err != nil {
		return err
	}
	sp := tracer.StartSpan("messageQueueHandler", opentracing.ChildOf(ctx))
	defer sp.Finish()

	req, err := jaegertracing.NewTracedRequest("PUT", "http://account:8080/account/booking/"+msg.Message, nil, sp)
	if err != nil {
		sp.SetTag("error", true)
		sp.LogKV("error.message", err.Error())
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		sp.SetTag("error", true)
		sp.LogKV("error.message", err.Error())
		return err
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("account service failed")
		sp.SetTag("error", true)
		sp.LogKV("error.message", err.Error())
		return err
	}

	return nil
}

func getSpanContext(ctx string) (opentracing.SpanContext, error) {
	h := opentracing.TextMapCarrier{
		jaeger.TraceContextHeaderName: ctx,
	}
	return tracer.Extract(opentracing.TextMap, h)
}
