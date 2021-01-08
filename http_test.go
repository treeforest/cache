package cache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

var scores_db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestHTTP(t *testing.T) {
	NewGroup("scores", 2 << 10, GetterFunc(
		func(key string) ([]byte, error) {
			t.Log("[SlowDB] search key: " + key)
			if v, ok := scores_db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:9999"

	go func() {
		peers := NewHTTPPool(addr)
		t.Log("cache is running at " + addr)
		t.Fatal(http.ListenAndServe(addr, peers))
	}()
	time.Sleep(time.Second)

	url := "http://" + addr + defaultBasePath + "scores/Tom"
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	val, _ := ioutil.ReadAll(resp.Body)
	if string(val) != "630" {
		t.Fatal("error")
	}

	url = "http://" + addr + defaultBasePath + "scores/kkk"
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	val, _ = ioutil.ReadAll(resp.Body)
	if !strings.Contains(string(val), "not exist") {
		t.Fatal("error")
	}
}