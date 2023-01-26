# jac : just another cache

jac is a persistent in-memory key:value/store cache based on go-cache that is basically am augmented thread-safe collections of `map[string]string` with expiration time called buckets.  
jac is suitable for application running on a single machine that need persistence and resistance towards system or application crashes. It achieved this by duplicating the data itself on the disk in a readable manner (optional) during execution and as a standard GOB once the application has closed. In this way it is possible to load a the latest cache state after an application restart or a system crash.  
jac generates two files either during operation (.data) or for shutdown back-up (.rec). The former is a JSON text file that is sued to reload the state in case of sudden system or application death. It can also be used to preload values into the cache.  The latter are GOB save files generated when the cache is properly closed and are the rpeferred mode to load the cache at application (re)start.  

jac has been compiled and tested for the following pairs:  

 - linux/arm  
 - linux/arm64  
 - linux/amd64  
 - openWRT/mipsle 
 - windows/amd64  

jac is built to accept data of any type, however the data will be always stored as a string and, to be type agnostic, will be also returned as a string.  

### Installation

`go get github.com/fpessolano/jac`

### Usage

```go
import (
	"fmt"
	"github.com/fpessolano/jac"
	"os"
)

func main() {
    // Initialise the cache. The options below are the standard values
    if err := jac.Initialise(true, nil); err != jac.IllegalParameter && err != nil {
        fmt.Println(err)
        os.Exit(0)
    }
	
    // Create a cache bucket called test with a default expiration time (see types.go)
    test, err := jac.NewBucket("definitions", jac.NoExpiration)
    if err != nil {
        fmt.Println(err)
        os.Exit(0)
    }

    // Set a persistent value of the key "one" to "1", with the default expiration time
    test.Set("one", 1, cache.DefaultExpiration, true)
    
    // Set a non-persistent value of the key "b" to "2", with no expiration time
    test.Set("two", "two", cache.NoExpiration, false)
    
    // Get the value associated with the key "one" from the cache
    oneRead, found := test.Get("one")
    if found {
        fmt.Println(oneRead)
    }
	
    // Get all items in bucket test
    allItems := test.Items()
    for _, el := range(allItems) {
        fmt.Print(el)	
    }
    
    // Close the bucket
    test.Close()
    
    // Close the cache
    jac.Terminate()
}
```

### Available methods

```go
    // Initialise prepare the cache for use. It accept two parameters:
    //  v : set to true for verbose (useful for development purposes)
    //  o : are the database options (see inm types.go for further details)
    func Initialise(v bool, o *Options) error
    
    // Terminate closes the cache and relative processes
    func Terminate()
    
    // NewBucket create a new bucket in the cache.
    //  If a rec file or a data file are present and are not older than time.Now() - maxage,
    //  they will be loaded in the cache
    func NewBucket(name string, exp time.Duration) (c Bucket, e error) 
    
    // Close closes a bucket storing values in the recovery data
    //  if keep is false the working data file will be deleted
    func (c *Bucket) Close(keep bool) 
    
    // Get read the value associated to the key k/
    //  It also returns false if the key has not value associated to it
    func (c *Bucket) Get(k string) (string, bool) 
    
    // GetWithExpiration performs the same operation as Get but it also returns the
    //  value expiration time
    func (c *Bucket) GetWithExpiration(k string) (string, time.Time, bool) 
    
    // Set writes a new key/value pair with a given expiration time t and
    //  It marks the key/value pair persistent if pers is true.
    func (c *Bucket) Set(k string, vn interface{}, t time.Duration, pers bool) 
    
    // Update updates the key/value pair with a given expiration time t and
    //  It marks the key/value pair persistent if pers is true. The working file is not changed
    //  even if pers is true
    func (c *Bucket) Update(k string, vn interface{}, t time.Duration, pers bool) 
    
    // Replace replaces an existing key/value pair only it already existing.
    //  It marks the key/value pair persistent if pers is true.
    func (c *Bucket) Replace(k string, vn interface{}, t time.Duration, pers bool) 
    
    // Add add a new key/value pair only if it does not already
    //  It marks the key/value pair persistent if pers is true.
    func (c *Bucket) Add(k string, vn interface{}, t time.Duration, pers bool) (string, bool) 
    
    // FunctionUpdate updates the new key/value with a custom function of type
    //  func(k, v string) (string, string). The function is given the actual key/value pair is present
    //  and key/"" otherwise and it expects a new key/value pair.
    //  if the function modofies the key, the old key/pair is deleted.
    //  It marks the key/value pair persistent if pers is true.
    func (c *Bucket) FunctionUpdate(k string, f updateFunc, t time.Duration, pers bool) (string, string, bool) 
    
    // Items returns all elements in the bucket as a map[string]string
    func (c *Bucket) Items() (rt map[string]string) 
    
    // Delete permanently removes an item from the bucket
    func (c *Bucket) Delete(k string) 
    
    // Compact initiate a compation request of the working files.
    //  Please note that all values will be used and any previously indicated persistence flag will be ignored.
    func (c *Bucket) Compact() 
    
    // Flush deletes all items from the bucket.
    func (c *Bucket) Flush() 
    
    // ItemCount returns the number of items in the bucket. This may include items that have
    //  expired, but have not yet been cleaned up.
    func (c *Bucket) ItemCount() int 
    
    // OnEvicted sets an (optional) function that is called with the key and value when an
    //  item is evicted from the bucketInternal. (Including when it is deleted manually, but
    //  not when it is overwritten.) Set to nil to disable.
    func (c *Bucket) OnEvicted(f func(string, interface{})) 
    
    // DeleteExpired deletes all expired items from the bucketInternal.
    func (c *Bucket) DeleteExpired() 
```

