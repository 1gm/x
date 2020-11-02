package log

import (
	"bytes"
	"time"

	"go.uber.org/zap/zapcore"
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
