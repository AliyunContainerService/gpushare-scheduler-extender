// Package colog implements prefix based logging by setting itself as output of the standard library
// and parsing the log messages. Level prefixes are called headers in CoLog terms to not confuse with
// log.Prefix() which is independent.
// Basic usage only requires registering:
//	func main() {
//		colog.Register()
//		log.Print("info: that's all it takes!")
//	}
//
// CoLog requires the standard logger to submit messages without prefix or flags. So it resets them
// while registering and assigns them to itself, unfortunately CoLog cannot be aware of any output
// previously set.
package colog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

// std is the global singleton
// analog of the standard log.std
var std = NewCoLog(os.Stderr, "", 0)

// CoLog encapsulates our log writer
type CoLog struct {
	mu           sync.Mutex
	host         string
	prefix       string
	minLevel     Level
	defaultLevel Level
	headers      HeaderMap
	extractor    Extractor
	formatter    Formatter
	customFmt    bool
	parseFields  bool
	fixed        Fields
	hooks        hookPool
	out          io.Writer
}

// Entry represents a message being logged and all attached data
type Entry struct {
	Level   Level     // severity: trace, debug, info, warning, error, alert
	Time    time.Time // time of the event
	Host    string    // host origin of the message
	Prefix  string    // Prefix set to the logger
	File    string    // file where the log was called
	Line    int       // line in the file where the log was called
	Message []byte    // logged message
	Fields  Fields    // map of key-value data parsed from the message
}

// Level represents severity level
type Level uint8

// LevelMap links levels with output header bytes
type LevelMap map[Level][]byte

// HeaderMap links input header strings with levels
type HeaderMap map[string]Level

// hookPool is a list of registered pool, grouped by Level
type hookPool map[Level][]Hook

// Fields is the key-value map for extracted data
type Fields map[string]interface{}

const (
	// Unknown severity level
	unknown Level = iota
	// LTrace represents trace severity level
	LTrace
	// LDebug represents debug severity level
	LDebug
	// LInfo represents info severity level
	LInfo
	// LWarning represents warning severity level
	LWarning
	// LError represents error severity level
	LError
	// LAlert represents alert severity level
	LAlert
)

// String implements the Stringer interface for levels
func (level Level) String() string {
	switch level {
	case LTrace:
		return "trace"
	case LDebug:
		return "debug"
	case LInfo:
		return "info"
	case LWarning:
		return "warning"
	case LError:
		return "error"
	case LAlert:
		return "alert"
	}

	return "unknown"
}

var initialMinLevel = LTrace
var initialDefaultLevel = LInfo

var defaultHeaders = HeaderMap{
	"t: ":       LTrace,
	"trc: ":     LTrace,
	"trace: ":   LTrace,
	"d: ":       LDebug,
	"dbg: ":     LDebug,
	"debug: ":   LDebug,
	"i: ":       LInfo,
	"inf: ":     LInfo,
	"info: ":    LInfo,
	"w: ":       LWarning,
	"wrn: ":     LWarning,
	"warn: ":    LWarning,
	"warning: ": LWarning,
	"e: ":       LError,
	"err: ":     LError,
	"error: ":   LError,
	"a: ":       LAlert,
	"alr: ":     LAlert,
	"alert: ":   LAlert,
	"panic: ":   LAlert,
}

// NewCoLog returns CoLog instance ready to be used in logger.SetOutput()
func NewCoLog(out io.Writer, prefix string, flags int) *CoLog {
	cl := new(CoLog)
	cl.minLevel = initialMinLevel
	cl.defaultLevel = initialDefaultLevel
	cl.hooks = make(hookPool)
	cl.fixed = make(Fields)
	cl.headers = defaultHeaders
	cl.prefix = prefix
	cl.formatter = &StdFormatter{Flag: flags}
	cl.extractor = &StdExtractor{}
	cl.SetOutput(out)
	if host, err := os.Hostname(); err != nil {
		cl.host = host
	}

	return cl
}

// Register sets CoLog as output for the default logger.
// It "hijacks" the standard logger flags and prefix previously set.
// It's not possible to know the output previously set, so the
// default os.Stderr is assumed.
func Register() {
	// Inherit standard logger flags and prefix if appropriate
	if !std.customFmt {
		std.formatter.SetFlags(log.Flags())
	}

	if log.Prefix() != "" && std.prefix == "" {
		std.SetPrefix(log.Prefix())
	}

	// Disable all extras
	log.SetPrefix("")
	log.SetFlags(0)

	// Set CoLog as output
	log.SetOutput(std)
}

// AddHook adds a hook to be fired on every event with
// matching level being logged. See the hook interface
func (cl *CoLog) AddHook(hook Hook) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	for _, l := range hook.Levels() {
		cl.hooks[l] = append(cl.hooks[l], hook)
	}
}

// SetHost sets the logger hostname assigned to the entries
func (cl *CoLog) SetHost(host string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.host = host
}

// SetPrefix sets the logger output prefix
func (cl *CoLog) SetPrefix(prefix string) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.prefix = prefix
}

// SetMinLevel sets the minimum level that will be actually logged
func (cl *CoLog) SetMinLevel(l Level) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.minLevel = l
}

// SetDefaultLevel sets the level that will be used when no level is detected
func (cl *CoLog) SetDefaultLevel(l Level) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.defaultLevel = l
}

// ParseFields activates or deactivates field parsing in the message
func (cl *CoLog) ParseFields(active bool) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.parseFields = active
}

// SetHeaders sets custom headers as the input headers to be search for to determine the level
func (cl *CoLog) SetHeaders(headers HeaderMap) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.headers = headers
}

// AddHeader adds a custom header to the input headers to be search for to determine the level
func (cl *CoLog) AddHeader(header string, level Level) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.headers[header] = level
}

// SetFormatter sets the formatter to use
func (cl *CoLog) SetFormatter(f Formatter) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.customFmt = true
	cl.formatter = f
}

// SetExtractor sets the formatter to use
func (cl *CoLog) SetExtractor(ex Extractor) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.extractor = ex
}

// FixedValue sets a key-value pair that will get automatically
// added to every log entry in this logger
func (cl *CoLog) FixedValue(key string, value interface{}) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.fixed[key] = value
}

// ClearFixedValues removes all previously set fields from the logger
func (cl *CoLog) ClearFixedValues() {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.fixed = make(Fields)
}

// Flags returns the output flags for the formatter if any
func (cl *CoLog) Flags() int {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if cl.formatter == nil {
		return 0
	}

	return cl.formatter.Flags()
}

// SetFlags sets the output flags for the formatter if any
func (cl *CoLog) SetFlags(flags int) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	if cl.formatter == nil {
		return
	}

	cl.formatter.SetFlags(flags)
}

// SetOutput is analog to log.SetOutput sets the output destination.
func (cl *CoLog) SetOutput(w io.Writer) {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	cl.out = w

	// if we have a color formatter, notify if new output supports color
	if _, ok := cl.formatter.(ColorFormatter); ok {
		cl.formatter.(ColorFormatter).ColorSupported(cl.colorSupported())
	}
}

// NewLogger returns a colog-enabled logger
func (cl *CoLog) NewLogger() *log.Logger {
	cl.mu.Lock()
	defer cl.mu.Unlock()

	return log.New(cl, "", 0)
}

// Write implements io.Writer interface to that the standard logger uses.
func (cl *CoLog) Write(p []byte) (n int, err error) {
	cl.mu.Lock()
	defer func() {
		cl.mu.Unlock()
		if r := recover(); r != nil {
			err = fmt.Errorf("error: colog: recovered panic %v\n", r)
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}()

	e := cl.parse(p)
	cl.extractFields(e)
	cl.fireHooks(e)

	if e.Level != unknown && e.Level < cl.minLevel {
		return 0, nil
	}

	if e.Level == unknown && cl.defaultLevel < cl.minLevel {
		return 0, nil
	}

	if cl.formatter == nil {
		err = errors.New("error: colog: missing formatter")
		fmt.Fprintln(os.Stderr, err.Error())
		return 0, err
	}

	fp, err := cl.formatter.Format(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: colog: failed to format entry: %v\n", err)
		return 0, err
	}

	n, err = cl.out.Write(fp)
	if err != nil {
		return n, err
	}

	return len(p), nil
}

func (cl *CoLog) parse(p []byte) *Entry {
	e := &Entry{
		Time:    time.Now(),
		Host:    cl.host,
		Prefix:  cl.prefix,
		Fields:  make(Fields),
		Message: bytes.TrimRight(p, "\n"),
	}

	// Apply fixed fields
	for k, v := range cl.fixed {
		e.Fields[k] = v
	}

	cl.applyLevel(e)

	// this is a bit expensive, check is anyone might actually need it
	if len(cl.hooks) != 0 || cl.formatter.Flags()&(log.Lshortfile|log.Llongfile) != 0 {
		e.File, e.Line = getFileLine(5)
	}

	return e
}

func (cl *CoLog) applyLevel(e *Entry) {
	for k, v := range cl.headers {
		header := []byte(k)
		if bytes.HasPrefix(e.Message, header) {
			e.Level = v                                     // apply level
			e.Message = bytes.TrimPrefix(e.Message, header) // remove header from message
			return
		}
	}

	e.Level = cl.defaultLevel
	return
}

// figure if output supports color
func (cl *CoLog) colorSupported() bool {

	// ColorSupporters can decide themselves
	if ce, ok := cl.out.(ColorSupporter); ok {
		return ce.ColorSupported()
	}

	// Windows users need ColorSupporter outputs
	if runtime.GOOS == "windows" {
		return false
	}

	// Check for Fd() method
	output, ok := cl.out.(interface {
		Fd() uintptr
	})

	// If no file descriptor it's not a TTY
	if !ok {
		return false
	}

	return isTerminal(int(output.Fd()))
}

func (cl *CoLog) extractFields(e *Entry) {
	if cl.parseFields && cl.extractor != nil {
		err := cl.extractor.Extract(e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: colog: failed to extract fields: %v\n", err)
		}
	}
}

func (cl *CoLog) fireHooks(e *Entry) {
	for k := range cl.hooks[e.Level] {
		err := cl.hooks[e.Level][k].Fire(e)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: colog: failed to fire hook: %v\n", err)
		}
	}
}

// Standard logger functions

// AddHook adds a hook to be fired on every event with
// matching level being logged on the standard logger
func AddHook(hook Hook) {
	std.AddHook(hook)
}

// SetHost sets the logger hostname assigned to the entries of the standard logger
func SetHost(host string) {
	std.SetHost(host)
}

// SetPrefix sets the logger output prefix of the standard logger
func SetPrefix(prefix string) {
	std.SetPrefix(prefix)
}

// SetMinLevel sets the minimum level that will be actually logged by the standard logger
func SetMinLevel(l Level) {
	std.SetMinLevel(l)
}

// SetDefaultLevel sets the level that will be used when no level is detected for the standard logger
func SetDefaultLevel(l Level) {
	std.SetDefaultLevel(l)
}

// ParseFields activates or deactivates field parsing in the message for the standard logger
func ParseFields(active bool) {
	std.ParseFields(active)
}

// SetHeaders sets custom headers as the input headers to be search for to determine the level for the standard logger
func SetHeaders(headers HeaderMap) {
	std.SetHeaders(headers)
}

// AddHeader adds a custom header to the input headers to be search for to determine the level for the standard logger
func AddHeader(header string, level Level) {
	std.AddHeader(header, level)
}

// Flags returns the output flags for the standard log formatter if any
func Flags() int {
	return std.Flags()
}

// SetFlags sets the output flags for the standard log formatter if any
func SetFlags(flags int) {
	std.SetFlags(flags)
}

// SetOutput is analog to log.SetOutput sets the output destination for the standard logger
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// SetFormatter sets the formatter to use by the standard logger
func SetFormatter(f Formatter) {
	std.SetFormatter(f)
}

// SetExtractor sets the extractor to use by the standard logger
func SetExtractor(ex Extractor) {
	std.SetExtractor(ex)
}

// FixedValue sets a field-value pair that will get automatically
// added to every log entry in the standard logger
func FixedValue(key string, value interface{}) {
	std.FixedValue(key, value)
}

// ClearFixedValues removes all previously set field-value in the standard logger
func ClearFixedValues() {
	std.ClearFixedValues()
}

// ParseLevel parses a string into a type Level
func ParseLevel(level string) (Level, error) {
	if lvl, ok := std.headers[level+": "]; ok {
		return lvl, nil
	}

	return unknown, fmt.Errorf("could not parse level %s", level)
}
