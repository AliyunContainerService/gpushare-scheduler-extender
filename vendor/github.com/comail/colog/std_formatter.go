package colog

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

var colorLabels = LevelMap{
	LTrace:   []byte("[ trace ] "),
	LDebug:   []byte("[ \x1b[0;36mdebug\x1b[0m ] "),
	LInfo:    []byte("[  \x1b[0;32minfo\x1b[0m ] "),
	LWarning: []byte("[  \x1b[0;33mwarn\x1b[0m ] "),
	LError:   []byte("\x1b[0;31m[ error ]\x1b[0m "),
	LAlert:   []byte("\x1b[0;37;41m[ alert ]\x1b[0m "),
}

var plainLabels = LevelMap{
	LTrace:   []byte("[ trace ] "),
	LDebug:   []byte("[ debug ] "),
	LInfo:    []byte("[  info ] "),
	LWarning: []byte("[  warn ] "),
	LError:   []byte("[ error ] "),
	LAlert:   []byte("[ alert ] "),
}

// StdFormatter supports plain and color level headers
// and bold/padded fields
type StdFormatter struct {
	mu             sync.Mutex
	Flag           int
	HeaderPlain    LevelMap
	HeaderColor    LevelMap
	Colors         bool // Force enable colors
	NoColors       bool // Force disable colors (has preference)
	colorSupported bool
}

// Format takes and entry and returns the formatted output in bytes
func (sf *StdFormatter) Format(e *Entry) ([]byte, error) {

	// Initialize if not set
	if sf.HeaderColor == nil {
		sf.mu.Lock()
		sf.HeaderColor = colorLabels
		sf.mu.Unlock()
	}

	// Initialize if not set
	if sf.HeaderPlain == nil {
		sf.mu.Lock()
		sf.HeaderPlain = plainLabels
		sf.mu.Unlock()
	}

	// Normal headers.  time, file, etc
	var header, message []byte
	sf.stdHeader(&header, e.Time, e.Prefix, e.File, e.Line)

	// Level headers
	headers := sf.levelHeaders()
	message = append(headers[e.Level], append(header, e.Message...)...)

	if e.Fields != nil {
		sf.stdFields(&message, e.Fields, e.Level)
	}

	return append(message, '\n'), nil
}

// levelHeaders returns plain or color level headers
// depending on user preference and output support
func (sf *StdFormatter) levelHeaders() LevelMap {
	switch {
	case sf.NoColors:
		return sf.HeaderPlain
	case sf.Colors:
		return sf.HeaderColor
	case sf.colorSupported:
		return sf.HeaderColor
	}
	return sf.HeaderPlain
}

// Flags returns the output flags for the formatter.
func (sf *StdFormatter) Flags() int {
	return sf.Flag
}

// SetFlags sets the output flags for the formatter.
func (sf *StdFormatter) SetFlags(flags int) {
	sf.Flag = flags
}

// ColorSupported enables or disables the colors, this will be called on every
func (sf *StdFormatter) ColorSupported(supp bool) {
	sf.Colors = supp
}

// Adapted replica of log.Logger.formatHeader
func (sf *StdFormatter) stdHeader(buf *[]byte, t time.Time, prefix, file string, line int) {
	*buf = append(*buf, prefix...)
	if sf.Flag&(log.Ldate|log.Ltime|log.Lmicroseconds) != 0 {
		if sf.Flag&log.Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}
		if sf.Flag&(log.Ltime|log.Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)
			if sf.Flag&log.Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}
	if sf.Flag&(log.Lshortfile|log.Llongfile) != 0 {
		if sf.Flag&log.Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}

		if sf.Colors {
			file = fmt.Sprintf("\x1b[1;30m%s:%d:\x1b[0m ", file, line)
		} else {
			file = fmt.Sprintf("%s:%d: ", file, line)
		}

		*buf = append(*buf, file...)
	}
}

func (sf *StdFormatter) stdFields(buf *[]byte, f Fields, level Level) {
	var keys []string
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var trail []byte
	tbuf := bytes.NewBuffer(trail)
	for _, k := range keys {
		if sf.Colors {
			fmt.Fprintf(tbuf, "  \033[1m%s\033[0m=%+v", k, f[k])
		} else {
			fmt.Fprintf(tbuf, "  %s=%+v", k, f[k])
		}
	}

	// Fields in right side of the screen if everything fits
	if sf.Colors {
		halfWidth := int(terminalWidth(int(os.Stderr.Fd())) / 2)
		if halfWidth > len(*buf) && halfWidth > len(tbuf.Bytes()) {
			// Add padding until half screen
			// 17 makes up to color escape characters
			h := halfWidth - len(*buf) + 17
			// Extra color escape characters in alert
			if level == LAlert {
				h = h + 3
			}

			// Apply padding
			for i := 0; i < h; i++ {
				*buf = append(*buf, ' ')
			}
		}
	}

	*buf = append(*buf, tbuf.Bytes()...)
}

// get file a line where logger was called
func getFileLine(calldepth int) (string, int) {

	var file string
	var line int

	var ok bool
	_, file, line, ok = runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}

	return file, line
}

// Replica of log.Logger.itoa
func itoa(buf *[]byte, i int, wid int) {
	var u = uint(i)
	if u == 0 && wid <= 1 {
		*buf = append(*buf, '0')
		return
	}

	// Assemble decimal in reverse order.
	var b [32]byte
	bp := len(b)
	for ; u > 0 || wid > 0; u /= 10 {
		bp--
		wid--
		b[bp] = byte(u%10) + '0'
	}
	*buf = append(*buf, b[bp:]...)
}
