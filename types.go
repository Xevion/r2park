package main

type Location struct {
	id      uint   // Used for registration internally
	name    string // Used for autocomplete & location selection
	address string // Not used in this application so far
}
