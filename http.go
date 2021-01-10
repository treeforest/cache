package cache

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/treeforest/cache/consistenthash"
	log "github.com/treeforest/logger"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const (
	defaultBasePath = "/_cache/"
	defaultReplicas = 50
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self        string
	basePath    string
	mu          sync.Mutex // guards peers and httpGetters
	peers       *consistenthash.Map
	httpGetters map[string]*httpGetter // keyed by e.g. "http://10.0.0.1;8001"
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Set updates the pool's list of peers.
func (pool *HTTPPool) Set(peers ...string) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.peers = consistenthash.New(defaultReplicas, nil)
	pool.peers.Add(peers...)
	pool.httpGetters = make(map[string]*httpGetter, len(peers))

	for _, peer := range peers {
		pool.httpGetters[peer] = &httpGetter{baseURL: peer + pool.basePath}
	}
}

// PickPeer picks a peer according to key
func (pool *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if peer := pool.peers.Get(key); peer != "" && peer != pool.self {
		pool.Log("Pick peer %s", peer)
		return pool.httpGetters[peer], true
	}

	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)

// Log info with server name
func (pool *HTTPPool) Log(format string, v ...interface{}) {
	log.Infof("[Server %s] %s", pool.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (pool *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, pool.basePath) {
		log.Fatal("HTTPPool serving unexpected path: " + r.URL.Path)
	}

	pool.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(pool.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "base request", http.StatusBadRequest)
		return
	}

	groupName, key := parts[0], parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: " + groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(view.ByteSlice())
}

type httpGetter struct {
	baseURL string
}

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	url := fmt.Sprintf("%v%v%v", h.baseURL, url.QueryEscape(group), url.QueryEscape(key))

	res, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "Get failed")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("server returned: %v", res.Status))
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "reading response body")
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)

