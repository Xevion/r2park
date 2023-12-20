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

// RegisterContext is used to store the contextual information used during registration.
type RegisterContext struct {
	hiddenKeys       []string // The form inputs that are hidden & unused
	requiredFormKeys []string // The form inputs that arne't hidden - required to submit the form
	propertyId       uint     // The property ID involved with the request
	residentId       uint     // The resident ID involved with the request
}

const (
	GuestCodeRequired    = iota
	GuestCodeNotRequired = iota
	Unknown              = iota
)
