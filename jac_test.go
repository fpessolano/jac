package jac

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"
)

type testData struct {
	Id    string `json:"id"`
	Flag1 bool   `json:"flag1"`
	Value int    `json:"value"`
	Flag2 bool   `json:"flag2"`
}

//goland:noinspection GoNilness
func Test_jac(t *testing.T) {
	const (
		Tries = 100
		Keep  = true
	)

	if err := Initialise(true, nil); err != IllegalParameter && err != nil {
		t.Errorf(err.Error())
	}
	println("create")
	bucket, e := NewBucket("test", 180*time.Second)
	if e != nil {
		t.Errorf(e.Error())
	}
	for i := 0; i < Tries; i++ {
		tmp := testData{
			Id:    strconv.Itoa(i),
			Flag1: false,
			Value: i,
			Flag2: false,
		}
		if msg, er := json.Marshal(tmp); er == nil {
			bucket.Set(strconv.Itoa(i), string(msg), 180*time.Second, true)
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)
	bucket.Close(true)

	time.Sleep(2 * time.Second)
	println("verify")
	bucket, e = NewBucket("test", 180*time.Second)
	if e != nil {
		t.Errorf(e.Error())
	}
	for i := 0; i < Tries; i++ {
		msg, found := bucket.Get(strconv.Itoa(i))
		if !found {
			t.Errorf("missing data")
		} else {
			var entry testData
			if err := json.Unmarshal([]byte(msg), &entry); err == nil {
				fmt.Println(entry)
				if entry.Value != i {
					t.Errorf("wrong data")
				}
			}
		}
	}
	time.Sleep(90 * time.Second)
	bucket.Close(Keep)

	Terminate()
	time.Sleep(2 * time.Second)

}
