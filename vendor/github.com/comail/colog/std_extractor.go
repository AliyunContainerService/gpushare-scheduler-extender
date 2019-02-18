package colog

import (
	"bytes"
	"regexp"
)

// regex to extract key-value (or quoted value) from the logged message
// if you can do this better please make a pull request
// this is just the result of lots of trial and error
var fieldsRegex = `(?P<key>([\pL0-9_]+))\s*=\s*((?P<value>([\pL0-9_]+))|(?P<quoted>("[^"]*"|'[^']*')))`

// StdExtractor implements a regex based extractor for key-value pairs
// both unquoted foo=bar and quoted foo="some bar" are supported
type StdExtractor struct {
	rxFields *regexp.Regexp
}

// Extract finds key-value pairs in the message and sets them as Fields
// in the entry removing the pairs from the message.
func (se *StdExtractor) Extract(e *Entry) error {
	if se.rxFields == nil {
		se.rxFields = regexp.MustCompile(fieldsRegex)
	}
	matches := se.rxFields.FindAllSubmatch(e.Message, -1)
	if matches == nil {
		return nil
	}

	var key, value []byte
	captures := make(map[string]interface{})

	// Look for positions with: fmt.Printf("%#v \n", rxFields.SubexpNames())
	// Will find positions []string{"", "key", "", "", "value", "", "quoted", ""}
	//                                    1               4            6

	for _, match := range matches {
		// First group, simple key-value detected
		if len(match[1]) > 0 && len(match[4]) > 0 {
			key, value = match[1], match[4]
		}

		// Second group, quoted value detected
		if len(match[1]) > 0 && len(match[6]) > 0 {
			key, value = match[1], match[6]
			value = value[1 : len(value)-1] // remove quotes, first and last character
		}

		captures[string(key)] = string(value)
	}

	if captures != nil {
		// Eliminate key=value from text and trim from the right
		e.Message = bytes.TrimRight(se.rxFields.ReplaceAll(e.Message, nil), " \n")
		for k, v := range captures {
			e.Fields[k] = v
		}
	}

	return nil
}
