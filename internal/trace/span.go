package trace

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Spaner struct {
	s trace.Span
}

func (s *Spaner) End(opts ...trace.SpanEndOption) {
	if s == nil {
		return
	}
	s.s.End(opts...)
}

func (s *Spaner) AddEvent(name string, options ...trace.EventOption) {
	if s == nil {
		return
	}
	s.s.AddEvent(name, options...)
}
func (s *Spaner) IsRecording() bool {
	if s == nil {
		return false
	}
	return s.s.IsRecording()
}
func (s *Spaner) RecordError(err error, options ...trace.EventOption) {
	if s == nil {
		return
	}
	s.s.RecordError(err, options...)
}
func (s *Spaner) SpanContext() trace.SpanContext {
	if s == nil {
		return trace.SpanContext{}
	}
	return s.s.SpanContext()
}
func (s *Spaner) SetStatus(code codes.Code, description string) {
	if s == nil {
		return
	}
	s.s.SetStatus(code, description)
}
func (s *Spaner) SetName(name string) {
	if s == nil {
		return
	}
	s.SetName(name)
}
func (s *Spaner) SetAttributes(kv ...attribute.KeyValue) {
	if s == nil {
		return
	}
	s.s.SetAttributes(kv...)
}
func (s *Spaner) TracerProvider() trace.TracerProvider {
	if s == nil {
		return nil
	}
	return s.s.TracerProvider()
}
