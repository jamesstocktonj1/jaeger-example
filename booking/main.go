package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/opentracing/opentracing-go"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/uber/jaeger-client-go"
)

type QueueEntry struct {
	SpanID  string `json:"span_id"`
	Message string `json:"message"`
}

var ch *amqp.Channel
var chName string

func main() {
	var err error

	ch, err = initRabbitMQ()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	s := echo.New()
	s.Use(middleware.Logger())
	s.Use(middleware.Recover())

	closer := jaegertracing.New(s, nil)
	defer closer.Close()

	g := s.Group("/booking")
	g.PUT("/queue/:id", queueHandler)

	s.Logger.Fatal(s.Start(":8080"))
}

func queueHandler(c echo.Context) error {
	sp := jaegertracing.CreateChildSpan(c, "Queue Handler")
	defer sp.Finish()

	req, err := jaegertracing.NewTracedRequest(echo.GET, "http://account:8080/account/login", nil, sp)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		sp.SetTag("error", true)
		sp.LogKV("error.message", err.Error())
		return echo.NewHTTPError(resp.StatusCode, "account service failed")
	}

	err = addToQueue(sp, c.Param("id"))
	if err != nil {
		return err
	}

	return c.JSON(200, map[string]string{"message": "queued"})
}

func addToQueue(psp opentracing.Span, id string) error {
	sp := opentracing.StartSpan("message_queue", opentracing.ChildOf(psp.Context()))
	defer sp.Finish()

	spanID := getSpanContext(sp)
	fmt.Printf("SpanID: %s\n", spanID)

	entry := QueueEntry{
		SpanID:  spanID,
		Message: id,
	}
	entryString, err := json.Marshal(entry)
	if err != nil {
		sp.SetTag("error", true)
		sp.LogKV("error.message", err.Error())
		return echo.NewHTTPError(echo.ErrInternalServerError.Code, "json marshal failed")
	}

	err = ch.Publish(
		"",     // exchange
		chName, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        entryString,
		})
	if err != nil {
		sp.SetTag("error", true)
		sp.LogKV("error.message", err.Error())
		return echo.NewHTTPError(echo.ErrInternalServerError.Code, "message queue failed")
	}
	return nil
}

func getSpanContext(sp opentracing.Span) string {
	h := opentracing.TextMapCarrier{}
	sp.Tracer().Inject(sp.Context(), opentracing.TextMap, h)

	return h[jaeger.TraceContextHeaderName]
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
