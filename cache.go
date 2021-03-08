package jac

import (
	"bufio"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Initialise prepare the cache for use. It accept two parameters:
//  v : set to true for verbose (useful for development purposes)
//  o : are the database options (see inm types.go for further details)
func Initialise(v bool, o *Options) error {

	verbose = v
	ex, err := os.Executable()
	if err != nil {
		return err
	}

	// set default values
	options = Options{
		ExpirationTime:     0,
		IntervalCompacting: 1440 * 60,
		InternalBuffering:  10,
		LoadDelayMs:        10,
		MaximumAge:         5 * 60,
		WorkingFolder:      filepath.Dir(ex) + "/",
		RecoveryFolder:     filepath.Dir(ex) + "/",
	}
	if o != nil {
		// read users values
		// 0 values are not allowed and will generate a non blocking error
		// values below a given minimum will be rejected
		err = IllegalParameter
		if o.MaximumAge > 0 {
			options.MaximumAge = o.MaximumAge
			err = nil
		}
		if o.IntervalCompacting >= 60 {
			options.IntervalCompacting = o.IntervalCompacting
			err = nil
		}
		if o.InternalBuffering >= 10 {
			options.InternalBuffering = o.InternalBuffering
			err = nil
		}
		if o.LoadDelayMs >= 5 {
			options.LoadDelayMs = o.LoadDelayMs
			err = nil
		}
		if o.ExpirationTime >= 0 {
			options.ExpirationTime = o.ExpirationTime
			err = nil
		}
		if o.WorkingFolder != "" {
			if err = os.MkdirAll(o.WorkingFolder, os.ModePerm); err != nil {
				fmt.Println(err)
				os.Exit(0)
			}
			options.WorkingFolder = o.WorkingFolder
			if options.WorkingFolder != "" && options.WorkingFolder[len(options.WorkingFolder)-1] != '/' {
				options.WorkingFolder += "/"
			}
		}
		if o.RecoveryFolder != "" {
			if err = os.MkdirAll(o.RecoveryFolder, os.ModePerm); err != nil {
				fmt.Println(err)
				os.Exit(0)
			}
			options.RecoveryFolder = o.RecoveryFolder
			if options.RecoveryFolder != "" && options.RecoveryFolder[len(options.RecoveryFolder)-1] != '/' {
				options.RecoveryFolder += "/"
			}
		}
	}

	// set internal processes and channels
	writeChannel = make(chan backupData, options.InternalBuffering)
	writeRstChannel = make(chan interface{})
	go writeHandler(nil)
	return nil
}

// Terminate closes the cache and relative processes
func Terminate() {
	writeRstChannel <- nil
	<-writeRstChannel
}

// NewBucket create a new bucket in the cache.
//  If a rec file or a data file are present and are not older than time.Now() - maxage,
//  they will be loaded in the cache
func NewBucket(name string, exp time.Duration) (c Bucket, e error) {
	c.name = name
	c.writer = writeChannel
	c.bucket = declare(time.Duration(options.ExpirationTime)*time.Second, time.Duration(2*options.ExpirationTime)*time.Second)
	c.cr = make(chan interface{})
	// first check for recovery file from normal termination
	statInfo, err := os.Stat(options.RecoveryFolder + name + ".rec")
	if !os.IsNotExist(err) {
		if (time.Now().Unix() - statInfo.ModTime().Unix()) < options.MaximumAge*60 {
			c.file, e = os.Create(options.WorkingFolder + name + ".data")
			if f, err := os.Open(options.RecoveryFolder + name + ".rec"); err == nil {
				var data map[string]Item
				dataDecoder := gob.NewDecoder(f)
				if err = dataDecoder.Decode(&data); err == nil {
					for i, v := range data {
						c.Set(i, fmt.Sprintf("%v", v.Object), exp, true)
					}
				} else {
					c.bucket = declare(time.Duration(options.ExpirationTime)*time.Second, time.Duration(2*options.ExpirationTime)*time.Second)
				}
				_ = f.Close()
			}
		}
		if e = os.Remove(options.RecoveryFolder + name + ".rec"); err != nil {
			return
		}
		go compactHandler(c)
		return
	}
	// check for working file remnants from server crash
	statInfo, err = os.Stat(options.WorkingFolder + name + ".data")
	if !os.IsNotExist(err) {
		if (time.Now().Unix() - statInfo.ModTime().Unix()) < options.MaximumAge*60 {

			if c.file, err = os.Open(options.WorkingFolder + name + ".data"); err == nil {
				scanner := bufio.NewScanner(c.file)
				for scanner.Scan() {
					var entry FileData
					if err := json.Unmarshal([]byte(scanner.Text()), &entry); err == nil {
						c.Update(entry.Key, entry.Value, exp, true)
					}
				}
				_ = c.file.Close()
			}
		}
		go compactHandler(c)
		return
	}
	c.file, e = os.Create(options.WorkingFolder + name + ".data")
	go compactHandler(c)
	return
}

// Close closes a bucket storing values in the recovery data
//  if keep is false the working data file will be deleted
func (c *Bucket) Close(keep bool) {
	if f, err := os.Create(options.RecoveryFolder + c.name + ".rec"); err == nil {
		// serialize the data
		dataEncoder := gob.NewEncoder(f)
		err = dataEncoder.Encode(c.bucket.bucketItems())
		_ = c.file.Close()
		_ = f.Close()
		if err == nil {
			if !keep {
				_ = os.Remove(options.WorkingFolder + c.name + ".data")
			}
		} else {
			_ = os.Remove(options.RecoveryFolder + c.name + ".rec")
		}
	} else {
		_ = c.file.Close()
	}
	c.file = nil
	go func() { c.cr <- nil }()
}

// Get read the value associated to the key k
//  It also returns false if the key has not value associated to it
func (c *Bucket) Get(k string) (string, bool) {
	c.bucket.mu.RLock()
	item, found := c.bucket.items[k]
	if !found {
		c.bucket.mu.RUnlock()
		return "", false
	}
	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.bucket.mu.RUnlock()
			return "", false
		}
	}
	c.bucket.mu.RUnlock()
	v := fmt.Sprintf("%v", item.Object)
	return v, found || v != ""
}

// GetWithExpiration performs the same operation as Get but it also returns the
//  value expiration time
func (c *Bucket) GetWithExpiration(k string) (string, time.Time, bool) {
	c.bucket.mu.RLock()
	item, found := c.bucket.items[k]
	if !found {
		c.bucket.mu.RUnlock()
		return "", time.Time{}, false
	}

	if item.Expiration > 0 {
		if time.Now().UnixNano() > item.Expiration {
			c.bucket.mu.RUnlock()
			return "", time.Time{}, false
		}

		// Return the item and the expiration time
		c.bucket.mu.RUnlock()
		return fmt.Sprintf("%v", item.Object), time.Unix(0, item.Expiration), true
	}

	// If expiration <= 0 (i.e. no expiration time set) then return the item
	// and a zeroed time.Time
	c.bucket.mu.RUnlock()
	return fmt.Sprintf("%v", item.Object), time.Time{}, true
}

// Set writes a new key/value pair with a given expiration time t and
//  It marks the key/value pair persistent if pers is true.
func (c *Bucket) Set(k string, vn interface{}, t time.Duration, pers bool) {
	if k == "" || vn == nil {
		return
	}
	v := anything2String(vn)
	if c.writer != nil && pers {
		select {
		case c.writer <- backupData{
			data: [2]string{k, v},
			file: c.file,
		}:
		case <-time.After(time.Duration(options.LoadDelayMs) * time.Millisecond):
		}
	}

	var e int64
	if t == DefaultExpiration {
		t = c.bucket.defaultExpiration
	}
	if t > 0 {
		e = time.Now().Add(t).UnixNano()
	}
	c.bucket.mu.Lock()
	c.bucket.items[k] = Item{
		Object:     v,
		Expiration: e,
	}
	c.bucket.mu.Unlock()
}

// Update updates the key/value pair with a given expiration time t and
//  It marks the key/value pair persistent if pers is true. The working file is not changed
//  even if pers is true
func (c *Bucket) Update(k string, vn interface{}, t time.Duration, pers bool) {
	c.bucket.mu.Lock()
	if k == "" || vn == nil {
		return
	}
	v := anything2String(vn)
	if _, found := c.bucket.get(k); found {
		c.bucket.set(k, v, t)
	} else {
		c.set(k, v, t, pers)
	}
	c.bucket.mu.Unlock()
}

// Replace replaces an existing key/value pair only it already existing.
//  It marks the key/value pair persistent if pers is true.
func (c *Bucket) Replace(k string, vn interface{}, t time.Duration, pers bool) {
	c.bucket.mu.Lock()
	if k == "" || vn == nil {
		return
	}
	v := anything2String(vn)
	if _, found := c.bucket.get(k); found {
		c.set(k, v, t, pers)
	}
	c.bucket.mu.Unlock()
}

// Add add a new key/value pair only if it does not already
//  It marks the key/value pair persistent if pers is true.
func (c *Bucket) Add(k string, vn interface{}, t time.Duration, pers bool) (string, bool) {
	c.bucket.mu.Lock()
	defer c.bucket.mu.Unlock()
	if k == "" || vn == nil {
		return "", false
	}
	v := anything2String(vn)
	if val, found := c.bucket.get(k); found && val != "" {
		return fmt.Sprintf("%v", val), true
	} else {
		c.set(k, v, t, pers)
		return "", false
	}
}

// FunctionUpdate updates the new key/value with a custom function of type
//  func(k, v string) (string, string). The function is given the actual key/value pair is present
//  and key/"" otherwise and it expects a new key/value pair.
//  if the function modofies the key, the old key/pair is deleted.
//  It marks the key/value pair persistent if pers is true.
func (c *Bucket) FunctionUpdate(k string, f updateFunc, t time.Duration, pers bool) (string, string, bool) {
	c.bucket.mu.Lock()
	defer c.bucket.mu.Unlock()
	if k == "" {
		return "", "", false
	}
	if val, found := c.bucket.get(k); found && val != nil {
		newK, newV := f(k, fmt.Sprintf("%v", val))
		if newK != k {
			c.bucket.delete(k)
		}
		c.set(newK, newV, t, pers)
		return newK, newV, true
	} else {
		newK, newV := f(k, "")
		c.set(newK, newV, t, pers)
		return newK, newV, false
	}
}

// Items returns all elements in the bucket as a map[string]string
func (c *Bucket) Items() (rt map[string]string) {
	rt = make(map[string]string)
	for i, v := range c.bucket.bucketItems() {
		if val := fmt.Sprintf("%v", v.Object); val != "" {
			rt[i] = val
		}
	}
	return
}

// Delete permanently removes an item from the bucket
func (c *Bucket) Delete(k string) {
	c.bucket.mu.Lock()
	v, evicted := c.bucket.delete(k)
	c.bucket.mu.Unlock()
	if evicted {
		c.bucket.onEvicted(k, v)
	}
}

// Compact initiate a compation request of the working files.
//  Please note that all values will be used and any previously indicated persistence flag will be ignored.
func (c *Bucket) Compact() {
	c.bucket.deleteExpired()
	select {
	case c.writer <- backupData{
		data: [2]string{},
		c:    c.bucket,
		file: c.file,
	}:
	default:
	}
}

// Flush deletes all items from the bucket.
func (c *Bucket) Flush() {
	c.bucket.mu.Lock()
	c.bucket.items = map[string]Item{}
	c.bucket.mu.Unlock()
}

// ItemCount returns the number of items in the bucket. This may include items that have
//  expired, but have not yet been cleaned up.
func (c *Bucket) ItemCount() int {
	c.bucket.mu.RLock()
	n := len(c.bucket.items)
	c.bucket.mu.RUnlock()
	return n
}

// OnEvicted sets an (optional) function that is called with the key and value when an
//  item is evicted from the bucketInternal. (Including when it is deleted manually, but
//  not when it is overwritten.) Set to nil to disable.
func (c *Bucket) OnEvicted(f func(string, interface{})) {
	c.bucket.mu.Lock()
	c.bucket.onEvicted = f
	c.bucket.mu.Unlock()
}

// DeleteExpired deletes all expired items from the bucketInternal.
func (c *Bucket) DeleteExpired() {
	c.bucket.deleteExpired()
}
