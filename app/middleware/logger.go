package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("token-bucket")

func RequestLogger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, span := tracer.Start(c.Request().Context(), c.Path())
		defer span.End()
		c.SetRequest(c.Request().WithContext(ctx))

		start := time.Now()
		err := next(c)
		latency := time.Since(start).Milliseconds()
		spanCtx := span.SpanContext()

		logrus.WithContext(ctx).WithFields(logrus.Fields{
			"method":     c.Request().Method,
			"path":       c.Path(),
			"status":     c.Response().Status,
			"latency_ms": latency,
			"remote_ip":  c.RealIP(),
			"trace_id":   spanCtx.TraceID().String(),
			"span_id":    spanCtx.SpanID().String(),
		}).Info("handled request")
		return err
	}
}
