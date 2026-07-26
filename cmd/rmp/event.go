package main

type event struct {
	etype eventType
	err   error // Only relevant for etype == eventError.
}

type eventType int

const (
	eventLoadingSong eventType = iota
	eventLoadingSimilar
	eventLoadedSimilar
	eventError
)
