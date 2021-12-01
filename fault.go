package fault

import (
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Fault interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

// decide decides if fault should be injected based on the provided ratio.
func decide(ratio float64) bool {
	mu.Lock()
	defer mu.Unlock()
	return r.Float64() < ratio
}

type Handler struct {
	f           Fault
	RandomRatio float64

	r  *rand.Rand
	mu sync.Mutex
}

func New(f Fault, randomRatio float64) *Handler {
	return &Handler{
		f:           f,
		RandomRatio: randomRatio,
		r:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (h *Handler) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Float64() < f.RandomRatio {
			next.ServeHTTP(w, r)
			return
		}

		h.f.Handle(w, R)
	})
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
}

// Handler adds delay to the given handler.
func (f *Delay) Handle(w http.ResponseWriter, r *http.Request) {
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
}

// Error injects arbitrary status code in the server call.
// Once this injection is enabled, the given error code is responded without
// calling actual server endpoint.
// You must initialize the struct before in use properly.
type Error struct {
	// StatusCode is the injected status code. Required.
	// This should be a valid HTTP status code, or Go's WriteHeader might cause panic.
	// Making sure setting the valid status code is the caller's responsibility.
	// While this struct is named Error, but technically setting 2xx code is OK and will work well.
	StatusCode int
	// StatusText is used as HTTP response body. Optional but if empty, a placeholder message is used.
	StatusText string
	// Random Ratio is the float64 number which is used to decide if delay should be added.
	// It should be between 0 and 1, but less than 0 or bigger than 1 does not give error.
	// Simply, if RandomRatio >= 1.0, then the delay injection rate will be 100%.
	RandomRatio float64
}

// Handler injects error to the given handler.
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

// DelayWithError combines Delay and Error into one.
// When this injection is enabled, it adds delay then respond an error. i.e.
// accepts the request -> sleep -> respond the given status code/text.
// There should be no actual server call.
type DelayWithError struct {
	// Duration defines how long the delay should be injected.
	Duration time.Duration
	// StatusCode is the injected status code. The same as the one in Error.
	StatusCode int
	// StatusText is the injected status text. The same as the one in Error.
	StatusText string
	// Random Ratio is the float64 number which is used to decide if delay should be added.
	// It should be between 0 and 1, but less than 0 or bigger than 1 does not give error.
	// Simply, if RandomRatio >= 1.0, then the delay injection rate will be 100%.
	RandomRatio float64
}

// Handler injects delay and error into the given handler
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

// Abort aborts the request.
// Internally it panics, and if it panics in Go, the HTTP request is interrupted and
// an empty response is returned.
// While it panics, stacktrace logging aren't shown in the server log.
type Abort struct {
	// Random Ratio is the float64 number which is used to decide if delay should be added.
	// It should be between 0 and 1, but less than 0 or bigger than 1 does not give error.
	// Simply, if RandomRatio >= 1.0, then the delay injection rate will be 100%.
	RandomRatio float64
}

// Handler aborts the request
func (f *Abort) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !decide(f.RandomRatio) {
			next.ServeHTTP(w, r)
			return
		}

		// If it panics with ErrAbortHandler in http handler, the server stacktrace logging will be suppressed.
		// https://pkg.go.dev/net/http#Handler
		panic(http.ErrAbortHandler)
	})
}

// DelayWithAbort aborts the request in the same way as Abort,
// the delay is injected before that.
// By default, it injects zero delay.
type DelayWithAbort struct {
	// Duration defines how long the delay should be injected.
	Duration time.Duration
	// Random Ratio is the float64 number which is used to decide if delay should be added.
	// It should be between 0 and 1, but less than 0 or bigger than 1 does not give error.
	// Simply, if RandomRatio >= 1.0, then the delay injection rate will be 100%.
	RandomRatio float64
}

// Handler adds delay and abort in the given handler
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
