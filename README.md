# jac : just another cache

jac is a persistent in-memory key:value/store cache based on go-cache that is basically am augmented thread-safe collections of `map[string]string` with expiration time called buckets.  
jac is suitable for application running on a single machine that need persistence and resistance towards system or application crashes. It achieved this by duplicating the data itself on the disk in a readable manner (optional) during execution and as a standard GOB once the application has closed. In this way it is possible to load a the latest cache state after an application restart or a system crash.  
jac generates two files either during operation (.data) or for shutdown back-up (.rec). The former is a JSON text file that is sued to reload the state in case of sudden system or application death. It can also be used to preload values into the cache.  The latter are GOB save files generated when the cache is properly closed and are the rpeferred mode to load the cache at application (re)start.  

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
    test.Set("one", "1", cache.DefaultExpiration, true)
    
    // Set a non-persistent value of the key "b" to "2", with no expiration time
    test.Set("two", "2", cache.NoExpiration, false)
    
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
