package cache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGetter(t *testing.T) {
	var f Getter = GetterFunc(func(key string) ([]byte, error) {
		return []byte(key), nil
	})

	expect := []byte("key")
	if v, _ := f.Get("key"); !reflect.DeepEqual(v, expect) {
		t.Errorf("callback failed")
	}
}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	g := NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			//t.Log("[SlowDB] search key " + key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := g.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		if _, err := g.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} else { // cache hit
			//t.Log("[Cache] hit")
		}
	}

	if view, err := g.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}

func createGroup() *Group {
	return NewGroup("scores", 2<<10, GetterFunc(
		func(key string) ([]byte, error) {
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, g *Group, t *testing.T) {
	peers := NewHTTPPool(addr)
	peers.Set(addrs...)
	g.RegisterPeers(peers)
	t.Log("cache is running at", addr)
	t.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, g *Group, t *testing.T) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := g.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))

	t.Log("fontend server is running at ", apiAddr)
	t.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

func http_get(url string, t *testing.T) string {
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	return string(b)
}

func TestGroup_GetFromRemote(t *testing.T) {
	api := true

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	for _, addr := range addrMap {
		group := createGroup()
		go startCacheServer(addr, []string(addrs), group, t)
		if api {
			go startAPIServer(apiAddr, group, t)
			api = false
			time.Sleep(time.Second)
		}
	}
	time.Sleep(time.Second)

	if http_get(apiAddr + "/api?key=Tom", t) != "630" {
		t.Fatal("Error value")
	}
	if !strings.Contains(http_get(apiAddr + "/api?key=kkk", t), "not exist") {
		t.Fatal("Error value")
	}
}