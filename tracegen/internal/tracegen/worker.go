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

func (w worker) setUpTracers() []trace.Tracer {
    toReturn := make([]trace.Tracer, 0, len(w.tracerProviders))

    for i := 0; i< len(w.tracerProviders); i++ {
        otel.SetTracerProvider(w.tracerProviders[i])
        tracer := otel.Tracer("tracegen"+string(i))
        toReturn = append(toReturn, tracer)
    }
    return toReturn
}

func (w worker) addChild(parentCtx context.Context, tracer trace.Tracer) context.Context {
    childCtx, child := tracer.Start(parentCtx, "okey-dokey", trace.WithAttributes(
        attribute.String("span.kind", "server"),
        semconv.NetPeerIPKey.String(fakeIP),
        semconv.PeerServiceKey.String("tracegen-client"),
    ))
    opt := trace.WithTimestamp(time.Now().Add(fakeSpanDuration))
    child.End(opt)
    return childCtx
}

func (w worker) simulateTraces() {
    // set up all tracers
    tracers := w.setUpTracers()
	limiter := rate.NewLimiter(w.limitPerSecond, 1)
	var i int
	for atomic.LoadUint32(w.running) == 1 {
		ctx, sp := tracers[0].Start(context.Background(), "lets-go", trace.WithAttributes(
			attribute.String("span.kind", "client"), // is there a semantic convention for this?
			semconv.NetPeerIPKey.String(fakeIP),
			semconv.PeerServiceKey.String("tracegen-server"),
		))

        //childCtx := w.addChild(ctx, tracers[1])
        //_ = w.addChild(childCtx, tracers[2])

        child1Ctx := w.addChild(ctx, tracers[1])
        w.addChild(ctx, tracers[1])
        w.addChild(child1Ctx, tracers[2])
        w.addChild(child1Ctx, tracers[3])
        w.addChild(child1Ctx, tracers[4])
        grandchild4Ctx := w.addChild(child1Ctx, tracers[5])
        w.addChild(grandchild4Ctx, tracers[6])
        w.addChild(grandchild4Ctx, tracers[7])
        w.addChild(grandchild4Ctx, tracers[8])
        w.addChild(grandchild4Ctx, tracers[9])
        w.addChild(grandchild4Ctx, tracers[10])
        /*
        w.addChild(grandchild4Ctx, tracers[11])
        */
		limiter.Wait(context.Background())

        opt := trace.WithTimestamp(time.Now().Add(fakeSpanDuration))
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
