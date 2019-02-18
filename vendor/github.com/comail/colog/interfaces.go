package colog

// Hook is the interface to be implemented by event hooks
type Hook interface {
	Levels() []Level   // returns the set of levels for which the hook should be triggered
	Fire(*Entry) error // triggers the hook, this function will be called for every eligible log entry
}

// Formatter interface must be implemented by message formatters
// Format(*Entry) will be called and the resulting bytes sent to output
type Formatter interface {
	Format(*Entry) ([]byte, error) // The actual formatter called every time
	SetFlags(flags int)            // Like the standard log.SetFlags(flags int)
	Flags() int                    // Like the standard log.Flags() int
}

// ColorFormatter interface can be implemented by formatters
// to get notifications on whether the output supports color
type ColorFormatter interface {
	Formatter
	ColorSupported(yes bool)
}

// ColorSupporter interface can be implemented by "smart"
// outputs that want to handle color display themselves
type ColorSupporter interface {
	ColorSupported() bool
}

// Extractor interface must be implemented by data extractors
// the extractor reads the message and tries to extract key-value
// pairs from the message and sets the in the entry
type Extractor interface {
	Extract(*Entry) error
}
