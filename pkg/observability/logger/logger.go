package logger

import (
	"context"
	"io"
	"log/slog"
)

type (
	loggerKey struct{}
)

func newProduction(w io.Writer) *slog.Logger {
	level := slog.LevelInfo
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func newDevelopment(w io.Writer) *slog.Logger {
	level := slog.LevelDebug
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

func New(w io.Writer, production bool) *slog.Logger {
	if production {
		return newProduction(w)
	}
	return newDevelopment(w)
}

func Context(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	logger := ctx.Value(loggerKey{})
	if logger == nil {
		logger = ctx.Value("logger")
		if logger == nil {
			return nil
		}
	}
	return logger.(*slog.Logger)
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
