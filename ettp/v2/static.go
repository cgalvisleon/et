package ettp

import (
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/middleware"
)

// entryFiles: the files that say which version is loaded; they are revalidated on each load (no-cache) and always
// proxied, so a page and a Module Federation manifest keep the origin of the gateway
var entryFiles = map[string]bool{
	"index.html":       true,
	"mf-manifest.json": true,
	"mf-stats.json":    true,
	"remoteEntry.js":   true,
}

// hashedRx: a file with a content hash in its name (index-BqzK3a1x.js), the same bytes forever
var hashedRx = regexp.MustCompile(`[-.][A-Za-z0-9_-]{8,}\.[A-Za-z0-9]+$`)

// staticHidden: what the browser sends that an origin of files never receives (the token and the cookies of the
// gateway's domain) or never sets on it
var staticHidden = []string{"Authorization", "Cookie"}

const (
	CacheNoCache   = "no-cache"
	CacheImmutable = "public, max-age=31536000, immutable"
)

/**
* isEntry: An entry file (entryFiles or any .html)
* @param file string
* @return bool
**/
func isEntry(file string) bool {
	name := path.Base(file)
	return entryFiles[name] || strings.HasSuffix(name, ".html")
}

/**
* wantsFallback: A page of a SPA: the prefix itself, or a path without an extension that the browser asks as html
* @param r *http.Request, rest string
* @return bool
**/
func wantsFallback(r *http.Request, rest string) bool {
	if rest == "" {
		return true
	}

	if path.Ext(rest) != "" {
		return false
	}

	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

/**
* cacheOf: The Cache-Control of a file: no-cache for an entry (or the fallback), immutable for a hashed file the
* origin answered without one; empty keeps the origin's
* @param file string, entry bool, res *http.Response
* @return string
**/
func cacheOf(file string, entry bool, res *http.Response) string {
	if entry {
		return CacheNoCache
	}

	if res.StatusCode == http.StatusOK && res.Header.Get("Cache-Control") == "" && hashedRx.MatchString(file) {
		return CacheImmutable
	}

	return ""
}

/**
* copyStream: Copies the body; an event stream is flushed at each read, so its events reach the browser as they come
* @param w http.ResponseWriter, res *http.Response
* @return error
**/
func copyStream(w http.ResponseWriter, res *http.Response) error {
	flusher, ok := w.(http.Flusher)
	stream := strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream")
	if !ok || !stream {
		_, err := io.Copy(w, res.Body)
		return err
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := res.Body.Read(buf)
		if n > 0 {
			if _, werr := w.Write(buf[:n]); werr != nil {
				return werr
			}
			flusher.Flush()
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

/**
* handlerStatic: Serves a file of a static route (TpStatic): the fallback for a page of a SPA, a 302 to the origin in
* redirect mode (not for an entry file) or the file proxied in streaming, without the token or the cookies, with the
* cache policy of cacheOf
* @params w http.ResponseWriter, r *http.Request
**/
func (s *Server) handlerStatic(w http.ResponseWriter, r *http.Request) {
	rw := &middleware.ResponseWriterWrapper{ResponseWriter: w}
	metric := middleware.GetMetrics(r)

	resolver, ok := r.Context().Value(ResoluteKey).(*Resolver)
	if !ok {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, "resolver not found")
		return
	}

	file := r.PathValue(Wildcard)
	target := resolver.URL
	fallback := resolver.Fallback != "" && wantsFallback(r, file)
	if fallback {
		file = resolver.Fallback
		target = strings.Replace(resolver.resolve, Wildcard, file, 1)
	}

	entry := fallback || isEntry(file)
	if resolver.Mode == ModeRedirect && !entry {
		http.Redirect(rw, r, target, http.StatusFound)
		s.HTTPSuccess(resolver, metric, rw)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, target, nil)
	if err != nil {
		s.HTTPError(resolver, metric, rw, r, http.StatusInternalServerError, err.Error())
		return
	}

	req.Header = r.Header.Clone()
	for _, name := range staticHidden {
		req.Header.Del(name)
	}

	res, err := s.client.Do(req)
	if err != nil {
		s.HTTPError(resolver, metric, rw, r, http.StatusBadGateway, err.Error())
		return
	}
	defer res.Body.Close()

	header := rw.Header()
	for key, values := range res.Header {
		if key == "Set-Cookie" {
			continue
		}
		header[key] = values
	}

	if cache := cacheOf(file, entry, res); cache != "" {
		header.Set("Cache-Control", cache)
	}

	rw.WriteHeader(res.StatusCode)
	if err := copyStream(rw, res); err != nil {
		// The status line is already sent: only the failure is recorded (as in handlerApi)
		logs.Errorf("error copying static response body: %s", err.Error())
		resolver.setStatus(TpStatusFailed)
		s.deleteRequest(resolver.ID)
		return
	}

	s.HTTPSuccess(resolver, metric, rw)
}
