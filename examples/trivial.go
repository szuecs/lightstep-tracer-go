// A trivial LightStep Go tracer example.
//
// $ go build -o lightstep_trivial github.com/lightstep/lightstep-tracer-go/examples/trivial
// $ ./lightstep_trivial --access_token=YOUR_ACCESS_TOKEN

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	lightstep "github.com/lightstep/lightstep-tracer-go"
	"github.com/lightstep/lightstep-tracer-go/lightstepgrpc"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

var accessToken = flag.String("access_token", "", "your LightStep access token")

func subRoutine(ctx context.Context) {
	trivialSpan, _ := opentracing.StartSpanFromContext(ctx, "test span")
	defer trivialSpan.Finish()
	trivialSpan.LogEvent("logged something")
	trivialSpan.LogFields(log.String("string_key", "some string value"), log.Object("trivialSpan", trivialSpan))

	subSpan := opentracing.StartSpan(
		"child span", opentracing.ChildOf(trivialSpan.Context()))
	trivialSpan.LogFields(log.Int("int_key", 42), log.Object("subSpan", subSpan),
		log.String("time.eager", fmt.Sprint(time.Now())),
		log.Lazy(func(fv log.Encoder) {
			fv.EmitString("time.lazy", fmt.Sprint(time.Now()))
		}))
	defer subSpan.Finish()
}

func main() {
	flag.Parse()
	if len(*accessToken) == 0 {
		fmt.Println("You must specify --access_token")
		os.Exit(1)
	}

	// Use LightStep as the global OpenTracing Tracer.

	grpcOption, err := lightstepgrpc.WithGRPC(lightstepgrpc.WithAddress("collector-staging.lightstep.com:443"))
	if err != nil {
		panic(err)
	}
	tracer := lightstep.NewTracer(*accessToken, grpcOption)
	defer tracer.Close(context.Background())

	opentracing.SetGlobalTracer(tracer)

	// Do something that's traced.
	subRoutine(context.Background())

	// Force a flush before exit.
	err = tracer.Flush(context.Background())
	if err != nil {
		panic(err)
	}
}
