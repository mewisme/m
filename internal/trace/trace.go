// Package trace provides lightweight context-keyed spans without OpenTelemetry.
package trace

import "context"

type spanKey struct{}

// Span is a named operation interval with optional attributes.
type Span interface {
	End()
	Attr(k, v string)
}

type span struct {
	name  string
	attrs map[string]string
	ended bool
	onEnd func(*span)
}

func (s *span) End() {
	if s == nil || s.ended {
		return
	}
	s.ended = true
	if s.onEnd != nil {
		s.onEnd(s)
	}
}

func (s *span) Attr(k, v string) {
	if s == nil || s.ended {
		return
	}
	if s.attrs == nil {
		s.attrs = make(map[string]string)
	}
	s.attrs[k] = v
}

type noopSpan struct{}

func (noopSpan) End()                {}
func (noopSpan) Attr(string, string) {}

// Start begins a span and stores it on the returned context.
func Start(ctx context.Context, name string) (context.Context, Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	s := &span{name: name}
	return context.WithValue(ctx, spanKey{}, s), s
}

// StartWithHook is like Start but invokes onEnd when the span ends (for debug reporters).
func StartWithHook(ctx context.Context, name string, onEnd func(name string, attrs map[string]string)) (context.Context, Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	s := &span{name: name}
	if onEnd != nil {
		s.onEnd = func(sp *span) { onEnd(sp.name, sp.attrs) }
	}
	return context.WithValue(ctx, spanKey{}, s), s
}

// SpanFrom returns the span on ctx, or a no-op span.
func SpanFrom(ctx context.Context) Span {
	if ctx == nil {
		return noopSpan{}
	}
	if s, ok := ctx.Value(spanKey{}).(Span); ok && s != nil {
		return s
	}
	return noopSpan{}
}

// Name returns the span name when ctx holds a *span; otherwise "".
func Name(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if s, ok := ctx.Value(spanKey{}).(*span); ok && s != nil {
		return s.name
	}
	return ""
}
