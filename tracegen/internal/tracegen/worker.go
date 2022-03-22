// Copyright The OpenTelemetry Authors
// Copyright (c) 2018 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tracegen // import "github.com/open-telemetry/opentelemetry-collector-contrib/tracegen/internal/tracegen"

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type worker struct {
	running          *uint32         // pointer to shared flag that indicates it's time to stop the test
	numTraces        int             // how many traces the worker has to generate (only when duration==0)
	propagateContext bool            // whether the worker needs to propagate the trace context via HTTP headers
	totalDuration    time.Duration   // how long to run the test for (overrides `numTraces`)
	limitPerSecond   rate.Limit      // how many spans per second to generate
	wg               *sync.WaitGroup // notify when done
	logger           *zap.Logger
    tracerProviders  []*sdktrace.TracerProvider
}

const (
	fakeIP string = "1.2.3.4"

	fakeSpanDuration = 123 * time.Microsecond
)

func (w worker) setUpTracers() {


}

func (w worker) simulateTraces() {
    // set up all tracers
    otel.SetTracerProvider(w.tracerProviders[1])
	tracer := otel.Tracer("tracegen")
	limiter := rate.NewLimiter(w.limitPerSecond, 1)
	var i int
	for atomic.LoadUint32(w.running) == 1 {
		ctx, sp := tracer.Start(context.Background(), "lets-go", trace.WithAttributes(
			attribute.String("span.kind", "client"), // is there a semantic convention for this?
			semconv.NetPeerIPKey.String(fakeIP),
			semconv.PeerServiceKey.String("tracegen-server"),
		))

		childCtx := ctx
		if w.propagateContext {
			header := propagation.HeaderCarrier{}
			// simulates going remote
			otel.GetTextMapPropagator().Inject(childCtx, header)

			// simulates getting a request from a client
			childCtx = otel.GetTextMapPropagator().Extract(childCtx, header)
		}

        otel.SetTracerProvider(w.tracerProviders[0])
	    tracerNew := otel.Tracer("new")

		grandchildCtx, child := tracerNew.Start(childCtx, "okey-dokey", trace.WithAttributes(
			attribute.String("span.kind", "server"),
			semconv.NetPeerIPKey.String(fakeIP),
			semconv.PeerServiceKey.String("tracegen-client"),
		))

		_, grandchild := tracerNew.Start(grandchildCtx, "iamgrandchild", trace.WithAttributes(
			attribute.String("span.kind", "server"),
			semconv.NetPeerIPKey.String(fakeIP),
			semconv.PeerServiceKey.String("tracegen-client"),
		))
		firstOpt := trace.WithTimestamp(time.Now().Add(fakeSpanDuration))
        grandchild.End(firstOpt)

        otel.SetTracerProvider(w.tracerProviders[1])

		_, child2 := tracer.Start(childCtx, "part3", trace.WithAttributes(
			attribute.String("span.kind", "server"),
			semconv.NetPeerIPKey.String(fakeIP),
			semconv.PeerServiceKey.String("tracegen-client"),
		))
		opt := trace.WithTimestamp(time.Now().Add(fakeSpanDuration))

		limiter.Wait(context.Background())

		child.End(opt)
		child2.End(opt)
		sp.End(opt)

		i++
		if w.numTraces != 0 {
			if i >= w.numTraces {
				break
			}
		}
	}
	w.logger.Info("traces generated", zap.Int("traces", i))
	w.wg.Done()
}
