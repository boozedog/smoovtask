package middleware

import "net/http"

// Chain applies middleware to a handler in the given order.
// The first middleware in the list wraps outermost (runs first).
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
