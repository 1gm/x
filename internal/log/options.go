package log

// Config is a type manipulated by Option functions.
type Config struct {
	// Level is the minimum level to log at, can be debug, info, warn, error or fatal.
	Level string
	// Format is the format to write logs in, can be json or text.
	Format string
	// OmitTimestamp will omit the 'ts' field from the log messages.
	OmitTimestamp bool
	// GlobalFields are key/value pairs that show up on every log message.
	GlobalFields map[string]string
	// OutputPaths is a list of file paths to write log outputs to.
	OutputPaths []string
	// ErrorOutputPaths is a list of file paths to write internal log error outputs to.
	ErrorOutputPaths []string
}

// Option is a function that mutates Config.
type Option func(*Config)

// WithOmitTimestamp sets the Config.OmitTimestamp property.
func WithOmitTimestamp() func(*Config) {
	return func(c *Config) {
		c.OmitTimestamp = true
	}
}

// WithFormat sets the Config.Format property, can be "json" or "text".
func WithFormat(fmt string) func(*Config) {
	return func(c *Config) {
		c.Format = fmt
	}
}

const (
	JSONFormat = "json"
	TextFormat = "text"
)

// WithLevel sets the Config.Level property, can be "debug", "info", "warn", "error", or "fatal".
func WithLevel(level string) func(*Config) {
	return func(c *Config) {
		c.Level = level
	}
}

const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	FatalLevel = "fatal"
)

var blackListFieldOptions = map[string]struct{}{
	"lvl": {},
	"msg": {},
	"ts":  {},
}

// WithFields appends a key value pair to the Config.GlobalFields.
func WithFields(key string, val string) func(*Config) {
	return func(c *Config) {
		if c != nil {
			if _, ok := blackListFieldOptions[key]; ok {
				return
			}

			if c.GlobalFields == nil {
				c.GlobalFields = make(map[string]string)
			}

			c.GlobalFields[key] = val
		}
	}
}

// WithOutputPaths sets the Config OutputPath property to configure
// where the logs are written to. Valid inputs include file paths and standard
// streams.
func WithOutputPaths(paths ...string) func(*Config) {
	return func(c *Config) {
		if c != nil {
			for _, path := range paths {
				if path != "" {
					c.OutputPaths = append(c.OutputPaths, path)
				}
			}
		}
	}
}

// WithErrorOutputPaths sets the Config ErrorOutputPath property to configure
// where the internal error logs are written to. Valid inputs include
// file paths and standard streams.
func WithErrorOutputPaths(paths ...string) func(*Config) {
	return func(c *Config) {
		if c != nil {
			for _, path := range paths {
				if path != "" {
					c.ErrorOutputPaths = append(c.ErrorOutputPaths, path)
				}
			}
		}
	}
}
