package log

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func init() {
	zap.ReplaceGlobals(New().Desugar())
}

// New creates a SugaredLogger, overriding any default options with the input options.
//
// The default options configure a text logger at the debug level. New will panic if any errors occur.
func New(options ...Option) *zap.SugaredLogger {
	const op = "log.New"

	cfg := &Config{
		Level:        DebugLevel,
		Format:       TextFormat,
		GlobalFields: make(map[string]string),
	}
	for _, opt := range options {
		opt(cfg)
	}

	// apply default output paths && error output if they aren't specified or if they were
	// invalid, e.g. nil.
	if cfg.OutputPaths == nil {
		WithOutputPaths("stderr")(cfg)
	}

	if cfg.ErrorOutputPaths == nil {
		WithErrorOutputPaths("stderr")(cfg)
	}

	if cfg.Level == "" {
		cfg.Level = DebugLevel
	}

	var lvl zap.AtomicLevel
	switch strings.ToLower(cfg.Level) {
	case InfoLevel:
		lvl = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case WarnLevel:
		lvl = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case ErrorLevel:
		lvl = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case FatalLevel:
		lvl = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	default:
		lvl = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	var enc string
	var timeEncoder zapcore.TimeEncoder
	var levelEncoder zapcore.LevelEncoder
	switch strings.ToLower(cfg.Format) {
	case JSONFormat:
		enc = "json"
		timeEncoder = zapcore.ISO8601TimeEncoder
		levelEncoder = zapcore.LowercaseLevelEncoder
	default:
		enc = "console"
		timeEncoder = consoleTimeEncoder
		levelEncoder = zapcore.CapitalLevelEncoder
	}

	encoderCfg := zapcore.EncoderConfig{
		MessageKey:       "msg",
		LevelKey:         "lvl",
		EncodeLevel:      levelEncoder,
		EncodeDuration:   zapcore.SecondsDurationEncoder,
		ConsoleSeparator: consoleSeparator,
	}

	if !cfg.OmitTimestamp {
		encoderCfg.EncodeTime = timeEncoder
		encoderCfg.TimeKey = "ts"
	}

	zapCfg := zap.Config{
		Level:            lvl,
		Encoding:         enc,
		OutputPaths:      cfg.OutputPaths,
		ErrorOutputPaths: cfg.ErrorOutputPaths,
		EncoderConfig:    encoderCfg,
	}

	lg, err := zapCfg.Build()
	if err != nil {
		panic(fmt.Errorf("%s: %v", op, err))
	}

	var globalFields []zap.Field
	for k, v := range cfg.GlobalFields {
		globalFields = append(globalFields, zap.String(k, v))
	}
	if len(cfg.GlobalFields) > 0 {
		lg = lg.With(globalFields...)
	}
	zap.ReplaceGlobals(lg)
	return lg.Sugar()
}

type contextKey int

const logContextKey contextKey = 0

type logContext map[interface{}]interface{}

// Put returns an keyvals annotated copy of the context
func Put(ctx context.Context, keyvals ...interface{}) context.Context {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "(MISSING)")
	}

	v := from(ctx)
	for i := 0; i < len(keyvals); i += 2 {
		v[keyvals[i]] = keyvals[i+1]
	}

	return context.WithValue(ctx, logContextKey, v)
}

func from(ctx context.Context) logContext {
	if v, ok := ctx.Value(logContextKey).(logContext); ok {
		return v
	}
	lc := make(map[interface{}]interface{})
	return lc
}

// With returns a Logger with annotated with the keyvals inside the context
func With(ctx context.Context) *zap.SugaredLogger {
	l := G()
	if ctx == nil {
		return l
	}

	vals := from(ctx)
	args := make([]interface{}, 0, len(vals))
	for k, v := range vals {
		args = append(args, k, v)
	}
	return l.With(args...)
}

// IfErr will log an error if fn returns one, annotated with one or more optional messages,
// each joined by a space preceding the error message. The logger used is the one returned
// by using From on ctx.
//	e.g. "postgres.UserStore could not find user with ID 1: inner err here"
func IfErr(ctx context.Context, fn func() error, msgs ...string) {
	if err := fn(); err != nil {
		var msg string
		if len(msgs) > 0 {
			msg = fmt.Sprintf("%s: %s", strings.Join(msgs, " "), err)
		} else {
			msg = err.Error()
		}
		With(ctx).Error(msg)
	}
}

// Debug takes an array of interface{} and concatenates them into a log message using the logger returned by From(ctx).
func Debug(ctx context.Context, args ...interface{}) {
	if len(args) > 0 {
		With(ctx).Debug(args...)
	}
}

// Debugf takes a template string and an array of interface{} and converts them
// into a log message, much like fmt.Errorf or fmt.Sprintf.
func Debugf(ctx context.Context, template string, args ...interface{}) {
	With(ctx).Debugf(template, args...)
}

// Debugw takes a message and an array of interface{} and logs a message using the keyvals
// as properties on the final logged object.
func Debugw(ctx context.Context, msg string, keyvals ...interface{}) {
	With(ctx).Debugw(msg, keyvals...)
}

// Info takes an array of interface{} and concatenates them into a log message using the logger returned by With(ctx).
func Info(ctx context.Context, args ...interface{}) {
	if len(args) > 0 {
		With(ctx).Info(args...)
	}
}

// Infof takes a template string and an array of interface{} and converts them
// into a log message, much like fmt.Errorf or fmt.Sprintf.
func Infof(ctx context.Context, template string, args ...interface{}) {
	With(ctx).Infof(template, args...)
}

// Infow takes a message and an array of interface{} and logs a message using the keyvals
// as properties on the final logged object.
func Infow(ctx context.Context, msg string, keyvals ...interface{}) {
	With(ctx).Infow(msg, keyvals...)
}

// Warn takes an array of interface{} and concatenates them into a log message using the logger returned by With(ctx).
func Warn(ctx context.Context, args ...interface{}) {
	if len(args) > 0 {
		With(ctx).Warn(args...)
	}
}

// Warnf takes a template string and an array of interface{} and converts them
// into a log message, much like fmt.Errorf or fmt.Sprintf.
func Warnf(ctx context.Context, template string, args ...interface{}) {
	With(ctx).Warnf(template, args...)
}

// Warnw takes a message and an array of interface{} and logs a message using the keyvals
// as properties on the final logged object.
func Warnw(ctx context.Context, msg string, keyvals ...interface{}) {
	With(ctx).Warnw(msg, keyvals...)
}

// Error takes an array of interface{} and concatenates them into a log message using the logger returned by With(ctx).
func Error(ctx context.Context, args ...interface{}) {
	if len(args) > 0 {
		With(ctx).Error(args...)
	}
}

// Errorf takes a template string and an array of interface{} and converts them
// into a log message, much like fmt.Errorf or fmt.Sprintf.
func Errorf(ctx context.Context, template string, args ...interface{}) {
	With(ctx).Errorf(template, args...)
}

// Errorw takes a message and an array of interface{} and logs a message using the keyvals
// as properties on the final logged object.
func Errorw(ctx context.Context, msg string, keyvals ...interface{}) {
	With(ctx).Errorw(msg, keyvals...)
}

func Println(args ...interface{}) {
	G().Info(args...)
}

func Printf(template string, args ...interface{}) {
	G().Infof(template, args...)
}

func StandardLog() *log.Logger {
	return zap.NewStdLog(G().Desugar())
}

func nilS() func() *zap.SugaredLogger {
	var nop = zap.NewNop().Sugar()
	return func() *zap.SugaredLogger {
		return nop
	}
}

var (
	// G represents the global logger
	G = zap.S

	// N represents a nil logger
	N = nilS()
)

const consoleSeparator = " | "

// ConsoleTimeEncoder serializes a time.Time to a floating-point number of seconds
// since the Unix epoch.
func consoleTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	var buf bytes.Buffer
	buf.WriteString(t.Format("2006-01-02"))
	buf.WriteString(consoleSeparator)
	buf.WriteString(t.Format("15:04:05.000"))
	enc.AppendString(buf.String())
}
