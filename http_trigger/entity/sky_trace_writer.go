package entity

import (
	"github.com/SkyAPM/go2sky"
	"time"
)

type SkyTraceWriter struct {
	span go2sky.Span
}

func NewSkyTraceWriter(span go2sky.Span) (SkyTraceWriter, error) {
	return SkyTraceWriter{span: span}, nil
}

func (s SkyTraceWriter) SendInfo(time time.Time, msg string) {
	s.span.Log(time, "INFO", msg)
}
func (s SkyTraceWriter) SendError(time time.Time, msg string) {
	s.span.Error(time, "ERROR", msg)
}

func (s SkyTraceWriter) End() {
	s.span.End()
}
