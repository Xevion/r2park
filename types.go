package main

type Location struct {
	id      uint   // Used for registration internally
	key     string // Used for registration form lookup
	name    string // Used for autocomplete & location selection
	address string // Not used in this application so far
}

type Field struct {
	text string // The text displayed
	id   string // The id of the field
}

const (
	GuestCodeRequired    = iota
	GuestCodeNotRequired = iota
	Unknown              = iota
)
