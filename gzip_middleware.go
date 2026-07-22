package main

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// gzipMiddleware compresses textual responses when the client advertises gzip
// support. The decision is delayed until the first write so handlers that rely
// on net/http's automatic Content-Type detection keep working as expected.
func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gzipWriter := &gzipResponseWriter{
			ResponseWriter: writer,
			request:        request,
			acceptsGzip:    requestAcceptsGzip(request),
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(gzipWriter, request)
		gzipWriter.close()
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	request     *http.Request
	acceptsGzip bool
	statusCode  int
	wroteHeader bool
	decided     bool
	compressing bool
	compressor  *gzip.Writer
}

func (writer *gzipResponseWriter) WriteHeader(statusCode int) {
	// net/http permits informational responses before the final response.
	if statusCode >= 100 && statusCode < 200 && statusCode != http.StatusSwitchingProtocols {
		writer.ResponseWriter.WriteHeader(statusCode)
		return
	}
	if writer.wroteHeader {
		return
	}

	writer.statusCode = statusCode
	writer.wroteHeader = true
}

func (writer *gzipResponseWriter) Write(content []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}

	if !writer.decided && len(content) > 0 {
		writer.start(content, true)
	}
	if !writer.decided {
		return len(content), nil
	}
	if writer.compressing {
		return writer.compressor.Write(content)
	}
	return writer.ResponseWriter.Write(content)
}

// Flush supports handlers that deliberately stream a textual response.
func (writer *gzipResponseWriter) Flush() {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if !writer.decided {
		writer.start(nil, false)
	}
	if writer.compressing {
		_ = writer.compressor.Flush()
	}
	if flusher, ok := writer.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (writer *gzipResponseWriter) ReadFrom(reader io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{writer}, reader)
}

func (writer *gzipResponseWriter) close() {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	if !writer.decided {
		// An empty response gains nothing from compression and, importantly,
		// must not acquire a gzip body for statuses such as 204 and 304.
		writer.start(nil, false)
	}
	if writer.compressor != nil {
		_ = writer.compressor.Close()
	}
}

func (writer *gzipResponseWriter) start(sample []byte, hasBody bool) {
	if writer.decided {
		return
	}
	writer.decided = true

	header := writer.Header()
	appendVary(header, "Accept-Encoding")
	if hasBody && header.Get("Content-Type") == "" {
		header.Set("Content-Type", http.DetectContentType(sample))
	}

	writer.compressing = hasBody &&
		writer.acceptsGzip &&
		responseCanHaveBody(writer.request, writer.statusCode) &&
		isCompressibleResponse(header)
	if writer.compressing {
		header.Set("Content-Encoding", "gzip")
		header.Del("Content-Length")
	}

	writer.ResponseWriter.WriteHeader(writer.statusCode)
	if writer.compressing {
		writer.compressor = gzip.NewWriter(writer.ResponseWriter)
	}
}

func requestAcceptsGzip(request *http.Request) bool {
	if request == nil {
		return false
	}

	values := request.Header.Values("Accept-Encoding")
	if len(values) == 0 {
		return false
	}

	gzipSeen := false
	gzipQuality := 0.0
	wildcardSeen := false
	wildcardQuality := 0.0
	for _, entry := range strings.Split(strings.Join(values, ","), ",") {
		parts := strings.Split(entry, ";")
		coding := strings.ToLower(strings.TrimSpace(parts[0]))
		if coding == "" {
			continue
		}

		quality := 1.0
		for _, parameter := range parts[1:] {
			name, value, found := strings.Cut(parameter, "=")
			if !found || !strings.EqualFold(strings.TrimSpace(name), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil || parsed < 0 || parsed > 1 {
				quality = 0
			} else {
				quality = parsed
			}
		}

		switch coding {
		case "gzip":
			gzipSeen = true
			if quality > gzipQuality {
				gzipQuality = quality
			}
		case "*":
			wildcardSeen = true
			if quality > wildcardQuality {
				wildcardQuality = quality
			}
		}
	}

	if gzipSeen {
		return gzipQuality > 0
	}
	return wildcardSeen && wildcardQuality > 0
}

func responseCanHaveBody(request *http.Request, statusCode int) bool {
	if request != nil && request.Method == http.MethodHead {
		return false
	}
	return statusCode >= 200 && statusCode != http.StatusNoContent && statusCode != http.StatusNotModified
}

func isCompressibleResponse(header http.Header) bool {
	if header.Get("Content-Encoding") != "" || header.Get("Content-Range") != "" {
		return false
	}
	if strings.Contains(strings.ToLower(header.Get("Content-Disposition")), "attachment") {
		return false
	}
	for _, directive := range strings.Split(header.Get("Cache-Control"), ",") {
		if strings.EqualFold(strings.TrimSpace(directive), "no-transform") {
			return false
		}
	}

	mediaType, _, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return false
	}
	mediaType = strings.ToLower(mediaType)
	if strings.HasPrefix(mediaType, "text/") || strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml") {
		return true
	}

	switch mediaType {
	case "application/json", "application/javascript", "application/x-javascript", "application/xml", "image/svg+xml":
		return true
	default:
		return false
	}
}

func appendVary(header http.Header, field string) {
	values := header.Values("Vary")
	for _, value := range values {
		for _, existing := range strings.Split(value, ",") {
			if strings.TrimSpace(existing) == "*" || strings.EqualFold(strings.TrimSpace(existing), field) {
				return
			}
		}
	}

	if len(values) == 0 {
		header.Set("Vary", field)
		return
	}
	header.Set("Vary", strings.Join(values, ", ")+", "+field)
}
