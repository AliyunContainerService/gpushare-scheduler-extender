[![Build Status](https://travis-ci.org/comail/colog.svg?branch=master)](https://travis-ci.org/comail/colog)&nbsp;[![godoc reference](https://godoc.org/comail.io/go/colog?status.png)](https://godoc.org/comail.io/go/colog)

# What's CoLog?

CoLog is a prefix-based leveled execution log for Go. It's heavily inspired by [Logrus](https://github.com/Sirupsen/logrus) and aims to offer similar features by parsing the output of the standard library log. If you don't understand what this means take a look at this picture.

![CoLog showcase](http://i.imgur.com/jx9pu1b.png)

## But why?

An introduction and the rationale behind CoLog can be found in this blog post: https://texlution.com/post/colog-prefix-based-logging-in-golang/

## Features

* Supports hooks to receive log entries and send them to external systems via `AddHook`
* Supports customs formatters (color/pain text and JSON built-in) via `SetFormatter`
* Provides 6 built-in levels: trace, debug, info, warning, error, alert
* Understands full, 3 letter, and 1 letter headers: `error:`, `err:`, `e:`
* Supports custom prefixes (headers in CoLog terms) via `SetHeaders` and `AddHeader`
* Control levels used via `SetMinLevel` and `SetDefaultLevel`
* Supports optionally parsing key=value or key='some value' pairs
* Supports custom key-value extractor via `SetExtractor`
* Supports permanent context values via `FixedValue` and `ClearFixedValues`
* Supports standalone loggers via `NewCoLog` and `NewLogger`
* Compatible with existing Logrus hooks and formatters via [cologrus](https://github.com/comail/cologrus)
* Supports Windows terminal colors via [wincolog](https://github.com/comail/wincolog)

## API stability

CoLog's API is very unlikely to get breaking changes, but there are no promises. That being said, CoLog only needs to be imported by final applications and if you have one of those, you should be vendoring you dependencies in the first place. CoLog has no external dependencies, to vendor it you just need to clone this repo anywhere you want and start using it.

## Usage examples

#### Basic usage

```go
package main

import (
	"log"

	"github.com/comail/colog"
)

func main() {
	colog.Register()
	log.Print("info: that's all it takes!")
}
```

#### JSON output to a file with field parsing

```go
package main

import (
	"log"
	"os"
	"time"

	"github.com/comail/colog"
)

func main() {
	file, err := os.OpenFile("temp_json.log", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		panic(err)
	}

	colog.Register()
	colog.SetOutput(file)
	colog.ParseFields(true)
	colog.SetFormatter(&colog.JSONFormatter{
		TimeFormat: time.RFC3339,
		Flag:       log.Lshortfile,
	})

	log.Print("info: logging this to json")
	log.Print("warning: with fields foo=bar")
}

// cat tempjson.log
// {"level":"info","time":"2015-08-16T13:26:07+02:00","file":"json_example.go","line":24,"message":"logging this to json"}
// {"level":"warning","time":"2015-08-16T13:26:07+02:00","file":"json_example.go","line":25,"message":"with fields","fields":{"foo":"bar"}}
```

#### Standalone logger with level control and fixed values

```go
package main

import (
	"log"
	"os"

	"github.com/comail/colog"
)

func main() {
	cl := colog.NewCoLog(os.Stdout, "worker ", log.LstdFlags)
	cl.SetMinLevel(colog.LInfo)
	cl.SetDefaultLevel(colog.LWarning)
	cl.FixedValue("worker_id", 42)

	logger := cl.NewLogger()
	logger.Print("this gets warning level")
	logger.Print("debug: this won't be displayed")
}

// [  warn ] worker 2015/08/16 13:43:06 this gets warning level    worker_id=42
```

#### Adding custom hooks

```go
package main

import (
	"fmt"
	"log"

	"github.com/comail/colog"
)

type myHook struct {
	levels []colog.Level
}

func (h *myHook) Levels() []colog.Level {
	return h.levels
}

func (h *myHook) Fire(e *colog.Entry) error {
	fmt.Printf("We got an entry: \n%#v", e)
	return nil
}

func main() {
	colog.Register()
	colog.ParseFields(true)

	hook := &myHook{
		levels: []colog.Level{
			colog.LInfo,    // the hook only receives
			colog.LWarning, // these levels
		},
	}

	colog.AddHook(hook)

	colog.SetMinLevel(colog.LError) // this affects only the output
	log.Print("info: something foo=bar")
}

// We got an entry:
// &colog.Entry{Level:0x3, Time:time.Time{sec:63575323196, nsec:244349216, loc:(*time.Location)(0x23f8c0)}, Host:"",
// Prefix:"", File:"/data/workspace/comail/comail/src/comail.io/go/colog/examples/hook_example.go", Line:37,
// Message:[]uint8{0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x69, 0x6e, 0x67}, Fields:colog.Fields{"foo":"bar"}}%
```
