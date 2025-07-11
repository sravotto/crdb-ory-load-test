package kratos

import "time"

var defaultDelay = 100 * time.Millisecond

type Identity struct {
	Email     string
	FirstName string
	LastName  string
}
