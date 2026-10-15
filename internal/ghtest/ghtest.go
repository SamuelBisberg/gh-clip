// Package ghtest provides a scriptable fake gh.REST for tests.
package ghtest

import (
	"encoding/json"
	"io"
	"sync"
)

// Call is one recorded request.
type Call struct {
	Method string
	Path   string
	Body   []byte
}

// Fake implements gh.REST by delegating to Handler and recording every call.
// Handler returns the JSON-encodable response value (or nil) and an error.
type Fake struct {
	mu      sync.Mutex
	Handler func(method, path string, body []byte) (any, error)
	Calls   []Call
}

// Do implements gh.REST.
func (f *Fake) Do(method, path string, body io.Reader, response any) error {
	var data []byte
	if body != nil {
		data, _ = io.ReadAll(body)
	}
	f.mu.Lock()
	f.Calls = append(f.Calls, Call{Method: method, Path: path, Body: data})
	f.mu.Unlock()

	out, err := f.Handler(method, path, data)
	if err != nil {
		return err
	}
	if response != nil && out != nil {
		raw, err := json.Marshal(out)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, response)
	}
	return nil
}
