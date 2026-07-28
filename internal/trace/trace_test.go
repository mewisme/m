package trace_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/trace"
)

func TestSpanLifecycle(t *testing.T) {
	ctx, sp := trace.Start(context.Background(), "fetch")
	sp.Attr("pkg", "left-pad")
	if trace.Name(ctx) != "fetch" {
		t.Fatalf("name=%q", trace.Name(ctx))
	}
	inner := trace.SpanFrom(ctx)
	inner.Attr("phase", "download")
	var ended bool
	ctx2, sp2 := trace.StartWithHook(context.Background(), "resolve", func(name string, attrs map[string]string) {
		ended = true
		if name != "resolve" {
			t.Errorf("name=%s", name)
		}
	})
	_ = ctx2
	sp2.End()
	if !ended {
		t.Fatal("onEnd not called")
	}
	sp.End()
	sp.End() // idempotent
}

func TestNoopSpanFromEmpty(t *testing.T) {
	trace.SpanFrom(context.Background()).Attr("k", "v")
	trace.SpanFrom(context.TODO()).End()
}
