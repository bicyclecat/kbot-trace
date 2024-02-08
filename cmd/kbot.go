/*
Copyright © 2023 NAME HERE
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	telebot "gopkg.in/telebot.v3"

	"github.com/hirosassa/zerodriver"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	// TeleToken bot topsecret data
	TeleToken = os.Getenv("TELE_TOKEN")
	// MetricsHost exporter host:port
	MetricsHost = os.Getenv("METRICS_HOST")
	// TracesHost exporter
	traceEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	tracer        trace.Tracer
	logger        = zerodriver.NewProductionLogger()
)

// Initialize OpenTelemetry
func initMetrics(ctx context.Context) {

	// Create a new OTLP Metric gRPC exporter with the specified endpoint and options
	metricExporter, _ := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(MetricsHost),
		otlpmetricgrpc.WithInsecure(),
	)

	otel.SetMeterProvider(
		sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(
				resource.NewWithAttributes(
					semconv.SchemaURL,
					semconv.ServiceNameKey.String(fmt.Sprintf("kbot_%s", appVersion)),
				),
			),
			sdkmetric.WithReader(
				// collects and exports metric data every 10 seconds.
				sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(10*time.Second)),
			),
		),
	)

	traceExporter, _ := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(traceEndpoint),
		otlptracehttp.WithInsecure(),
	)

	// Create a new tracer provider with a batch span processor and the given exporter.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(fmt.Sprintf("kbot_%s", appVersion)),
			),
		),
	)

	// Handle shutdown properly so nothing leaks.
	defer func() { _ = tp.Shutdown(ctx) }()

	otel.SetTracerProvider(tp)

	// Finally, set the tracer that can be used for this package.
	tracer = tp.Tracer("kbot-trace")

}

func pmetrics(ctx context.Context, payload string) {
	// Get the global MeterProvider and create a new Meter with the name "kbot_light_signal_counter"
	meter := otel.GetMeterProvider().Meter("kbot_light_signal_counter")

	// Get or create an Int64Counter instrument with the name "kbot_light_signal_<payload>"
	counter, _ := meter.Int64Counter(fmt.Sprintf("kbot_light_signal_%s", payload))

	// Add a value of 1 to the Int64Counter
	counter.Add(ctx, 1)
}

// kbotCmd represents the kbot command
var kbotCmd = &cobra.Command{
	Use:     "kbot",
	Aliases: []string{"start"},
	Short:   "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		kbot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  TeleToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			logger.Fatal().Str("Error", err.Error()).Msg("Please check TELE_TOKEN")
			return
		} else {
			logger.Info().Str("Version", appVersion).Msg("kbot started")

		}

		trafficSignal := make(map[string]map[string]int8)

		trafficSignal["red"] = make(map[string]int8)
		trafficSignal["amber"] = make(map[string]int8)
		trafficSignal["green"] = make(map[string]int8)

		trafficSignal["red"]["pin"] = 12
		trafficSignal["amber"]["pin"] = 27
		trafficSignal["green"]["pin"] = 22

		kbot.Handle(telebot.OnText, func(m telebot.Context) error {
			ctx, span := tracer.Start(context.Background(), "kbot-message-processing")

			logger.Info().Str("Payload", m.Text()).Msg(m.Message().Payload)

			payload := m.Message().Payload

			// Add kbot message to span
			span.SetAttributes(attribute.String("telegram.message.text", m.Text()))
			defer span.End()

			pmetrics(ctx, payload)

			switch payload {
			case "hello":
				err = m.Send(fmt.Sprintf("Hello I'm Kbot %s!", appVersion))

			case "time":
				// Get current time and date
				currentTime := time.Now().Format("2006-01-02 15:04:05")
				err = m.Send(fmt.Sprintf("Current time and date: %s", currentTime))

			case "red", "amber", "green":

				if trafficSignal[payload]["on"] == 0 {
					trafficSignal[payload]["on"] = 1
				} else {
					trafficSignal[payload]["on"] = 0
				}

				err = m.Send(fmt.Sprintf("Switch %s light signal to %d", payload, trafficSignal[payload]["on"]))

			default:
				err = m.Send("Usage: /s red|amber|green")

			}

			return err

		})

		kbot.Start()
	},
}

func init() {
	ctx := context.Background()
	initMetrics(ctx)

	rootCmd.AddCommand(kbotCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kbotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kbotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Initialize OpenTelemetry tracer

}
