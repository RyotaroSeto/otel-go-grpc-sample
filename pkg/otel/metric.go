package otel

// func initMeterProvider() *metric.MeterProvider {
// 	ctx := context.Background()

// 	exporter, err := otlpmetricgrpc.New(ctx)
// 	if err != nil {
// 		log.Fatalf("new otlp metric grpc exporter failed: %v", err)
// 	}

// 	mp := metric.NewMeterProvider(
// 		metric.WithReader(metric.NewPeriodicReader(exporter)),
// 		metric.WithResource(initResource()),
// 	)
// 	otel.SetMeterProvider(mp)
// 	return mp
// }

// mp := initMeterProvider()
// defer func() {
// 	if err := mp.Shutdown(context.Background()); err != nil {
// 		log.Fatalf("Error shutting down meter provider: %v", err)
// 	}
// 	log.Println("Shutdown meter provider")
// }()
// openfeature.SetProvider(flagd.NewProvider())

// err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
// if err != nil {
// 	log.Fatal(err)
// }
