package colog

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"sync"
)

// JSONFormatter serializes entries to JSON
// TimeFormat can be any Go time format, if empty
// it will mimic the standard logger format
// LevelAsNum will use a numeric string "1", "2",...
// for as levels instead of "trace", "debug", ..
type JSONFormatter struct {
	mu         sync.Mutex
	TimeFormat string
	LevelAsNum bool
	Flag       int
}

// JSONEntry is an entry with the final JSON field types
// We can not just implement the Marshaller interface since
// some of the process depends on runtime options
type JSONEntry struct {
	Level   string `json:"level,omitempty"`
	Time    string `json:"time,omitempty"`
	Host    string `json:"host,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message,omitempty"`
	Fields  Fields `json:"fields,omitempty"`
}

// Format takes and entry and returns the formatted output in bytes
func (jf *JSONFormatter) Format(e *Entry) ([]byte, error) {

	file, line := jf.fileLine(e)
	date := jf.date(e)

	var level string
	if jf.LevelAsNum {
		level = strconv.Itoa(int(e.Level))
	} else {
		level = e.Level.String()
	}

	je := &JSONEntry{
		Level:   level,
		Time:    date,
		Host:    e.Host,
		Prefix:  e.Prefix,
		File:    file,
		Line:    line,
		Message: string(e.Message),
		Fields:  e.Fields,
	}

	data, err := json.Marshal(je)
	return append(data, '\n'), err
}

// Flags returns the output flags for the formatter.
func (jf *JSONFormatter) Flags() int {
	return jf.Flag
}

// SetFlags sets the output flags for the formatter.
func (jf *JSONFormatter) SetFlags(flags int) {
	jf.Flag = flags
}

func (jf *JSONFormatter) fileLine(e *Entry) (file string, line int) {
	if jf.Flag&(log.Lshortfile|log.Llongfile) == 0 {
		return
	}

	file = e.File
	line = e.Line
	if jf.Flag&log.Lshortfile != 0 {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
	}

	return file, line
}

func (jf *JSONFormatter) date(e *Entry) (date string) {
	if jf.TimeFormat != "" {
		return e.Time.Format(jf.TimeFormat)
	}

	if jf.Flag&(log.Ldate|log.Ltime|log.Lmicroseconds) == 0 {
		return ""
	}

	if jf.Flag&log.Ldate != 0 {
		year, month, day := e.Time.Date()
		date = fmt.Sprintf("%d/%d/%d", year, month, day)
	}

	if jf.Flag&(log.Ltime|log.Lmicroseconds) != 0 {
		hour, min, sec := e.Time.Clock()
		date = fmt.Sprintf("%s %d:%d:%d", date, hour, min, sec)
		if jf.Flag&log.Lmicroseconds != 0 {
			date = fmt.Sprintf("%s.%d", date, e.Time.Nanosecond())

		}
	}

	return date
}
