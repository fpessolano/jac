package jac

import (
	"os"
	"sync"
	"time"
)

// external data types

// Default options are
//	ExpirationTime:     0,
//	IntervalCompacting: 1440 * 60,
//	InternalBuffering:  10,
//	LoadDelayMs:        10,
//	MaximumAge:         5 * 60,
type Options struct {
	ExpirationTime     int    // Expiration time is seconds
	IntervalCompacting int    // Cache working files compacting interval in seconds (values smaller than 60s will be defaulted to 60s)
	InternalBuffering  int    // Buffering length to decouple the in-memory cache from the disk processes. Bigger numbers improve cache speed at expenses of system crash resistance
	LoadDelayMs        int    // Regulates the start-up delay. Smaller numbers improves start-up time at costs of possible loss of persistence
	MaximumAge         int64  // Maximum age (in s) of a back-up file (.rec) or working file (.data) for it to be used to initialise the cache
	WorkingFolder      string // folder for working files (.data). File contain the entire cache in a readable. Altering the files only affects the initial cache load not its operation
	RecoveryFolder     string // folder for back-up files (.rec)
}

type Item struct {
	Object     interface{}
	Expiration int64
}

type FileData struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Bucket struct {
	name   string
	bucket *bucketInternalPtr
	file   *os.File
	writer chan backupData
	cr     chan interface{}
}

// internal data types
type bucketInternalPtr struct {
	*bucketInternal
}

type bucketInternal struct {
	defaultExpiration time.Duration
	items             map[string]Item
	mu                sync.RWMutex
	onEvicted         func(string, interface{})
	janitor           *janitor
}

type keyAndValue struct {
	key   string
	value interface{}
}

type janitor struct {
	Interval time.Duration
	stop     chan bool
}

type backupData struct {
	data [2]string
	c    *bucketInternalPtr // when not nil a file compaction is requested
	file *os.File
}

type updateFunc func(k, v string) (string, string)
