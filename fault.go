package fault

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

var r *rand.Rand
var mu sync.Mutex

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// decide decides if fault should be injected based on the provided ratio.
func decide(ratio float64) bool {
	mu.Lock()
	defer mu.Unlock()
	return r.Float64() < ratio
}

// Delay injects delay in the server call.
// This can be used to simulate slow network.
// You must initialize the struct before in use properly; If you use it with zero values,
// the delay won't be added by default.
type Delay struct {
	// Duration defines how long the delay should be injected.
	Duration time.Duration
	// Afterward defines where delay should be injected in the Handler process.
	// If true, the delay is added after server call; request comes in, proxied to next, sleep, then return response.
	// This is used to simulate slow network at response time.
	// For example, you can use it to make sure your server's idempotency.
	// If false, the delay is added before server call; request comes in, sleep, proxied to next, return response.
	Afterward bool
	// Random Ratio is the float64 number which is used to decide if delay should be added.
	// It should be between 0 and 1, but less than 0 or bigger than 1 does not give error.
	// Simply, if RandomRatio >= 1.0, then the delay injection rate will be 100%.
	RandomRatio float64
}

// Handler adds delay to  the given handler.
func (f *Delay) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !decide(f.RandomRatio) {
			next.ServeHTTP(w, r)
			return
		}

		// If Afterward is true, proxy -> sleep
		if f.Afterward {
			next.ServeHTTP(w, r)
			time.Sleep(f.Duration)
			return
		}

		// else, sleep -> proxy
		time.Sleep(f.Duration)
		next.ServeHTTP(w, r)
	})
}

type Error struct {
	StatusCode  int
	StatusText  string
	RandomRatio float64
}

func (f *Error) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !decide(f.RandomRatio) {
			next.ServeHTTP(w, r)
			return
		}

		statusText := f.StatusText
		if statusText == "" {
			statusText = "fault: pseudo status text is injected"
		}

		w.WriteHeader(f.StatusCode)
		w.Write([]byte(statusText))
	})
}

type DelayWithError struct {
	Duration    time.Duration
	StatusCode  int
	StatusText  string
	RandomRatio float64
}

func (f *DelayWithError) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !decide(f.RandomRatio) {
			next.ServeHTTP(w, r)
			return
		}

		statusText := f.StatusText
		if statusText == "" {
			statusText = "fault: pseudo status text is injected"
		}

		time.Sleep(f.Duration)
		w.WriteHeader(f.StatusCode)
		w.Write([]byte(statusText))
	})
}

type Abort struct {
	RandomRatio float64
}

func (f *Abort) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// https://pkg.go.dev/net/http#Handler
		panic(http.ErrAbortHandler)
	})
}

type DelayWithAbort struct {
	Duration    time.Duration
	RandomRatio float64
}

func (f *DelayWithAbort) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !decide(f.RandomRatio) {
			next.ServeHTTP(w, r)
			return
		}

		time.Sleep(f.Duration)
		// https://pkg.go.dev/net/http#Handler
		panic(http.ErrAbortHandler)
	})
}
