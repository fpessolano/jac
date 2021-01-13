package jac

import "time"

const (
	// To be used to set key/pair without expiration date
	NoExpiration time.Duration = -1
	// To be used to set key/pair with expiration date as from the bucket declaration
	DefaultExpiration time.Duration = 0
)

var options Options
var writeChannel chan backupData
var writeRstChannel chan interface{}
var verbose bool
