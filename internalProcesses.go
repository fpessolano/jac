package jac

import (
	"encoding/json"
	"fmt"
	"time"
)

// working file write handler
func writeHandler(consolidateTimers map[string]int64) {
	defer func() {
		if r := recover(); r != nil {
			writeHandler(consolidateTimers)
		}
	}()
	if consolidateTimers == nil {
		consolidateTimers = make(map[string]int64)
	}
	if verbose {
		println("start writeHandler")
	}
	for {
		select {
		case <-writeRstChannel:
			if verbose {
				fmt.Println("writeHandler closing")
			}
			writeRstChannel <- nil
		case nw := <-writeChannel:
			//fmt.Println("received", nw)
			if nw.file != nil {
				if nw.c != nil {
					name := fmt.Sprintf("%v", nw.file.Name())
					tm, skip := consolidateTimers[name]
					if skip {
						skip = time.Now().Unix()-tm < int64(options.IntervalCompacting)
					} else {
						consolidateTimers[name] = time.Now().Unix() - 1
					}
					if !skip {
						// consolidation
						if e := nw.file.Truncate(0); e == nil {
							if _, e = nw.file.Seek(0, 0); e == nil {
								for i, v := range nw.c.bucketItems() {
									if fmt.Sprintf("%v", v.Object) != "" {
										if data, err := json.Marshal(FileData{
											Key:   i,
											Value: fmt.Sprintf("%v", v.Object),
										}); err == nil {
											_, _ = nw.file.WriteString(string(data) + "\n")
										}
									}
								}
							}
						}
						continue
					}
				}
				// Update
				if data, err := json.Marshal(FileData{
					Key:   nw.data[0],
					Value: nw.data[1],
				}); err == nil {
					_, _ = nw.file.WriteString(string(data) + "\n")
				}
			}
		}
	}
}

// working file compact handler
func compactHandler(c Bucket) {
	defer func() {
		if r := recover(); r != nil {
			compactHandler(c)
		}
	}()
	if verbose {
		println("start compactHandler for " + c.name)
	}
	for {
		select {
		case <-c.cr:
			if verbose {
				fmt.Println("compactHandler closing for", c.name)
			}
		case <-time.After(time.Duration(options.IntervalCompacting) * time.Second):
			// in case of a zombie
			if c.file == nil {
				return
			}
			c.bucket.deleteExpired()
			c.writer <- backupData{
				c:    c.bucket,
				file: c.file,
			}
		}
	}

}
