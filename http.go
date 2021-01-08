package cache

import (
	"fmt"
	log "github.com/treeforest/logger"
	"net/http"
	"strings"
)

const defaultBasePath = "/_cache/"

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:80000"
	self     string
	basePath string
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

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
