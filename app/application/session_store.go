package application

import (
	"context"
	"net/http"

	"github.com/inertia-go/inertia-go/session"
)

type messagesKey struct{}
type requestMessages struct {
	errors map[string]string
	flash  map[string]any
}

func messagesFrom(r *http.Request) *requestMessages {
	if state, ok := r.Context().Value(messagesKey{}).(*requestMessages); ok {
		return state
	}
	return &requestMessages{}
}

// SessionContext must wrap the native adapter middleware. Its per-request holder
// captures consumed values so application Render can preserve named error bags
// and Always semantics without changing the upstream adapter source.
func (a *Application) SessionContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := &requestMessages{}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), messagesKey{}, state)))
	})
}

type captureStore struct{ store *session.CookieStore }

func (s *captureStore) FlashErrors(w http.ResponseWriter, r *http.Request, bag string, errs map[string]string) error {
	return s.store.FlashErrors(w, r, bag, errs)
}
func (s *captureStore) TakeErrors(w http.ResponseWriter, r *http.Request, bag string) (map[string]string, error) {
	errors, err := s.store.TakeErrors(w, r, bag)
	messagesFrom(r).errors = errors
	return errors, err
}
func (s *captureStore) FlashMessage(w http.ResponseWriter, r *http.Request, key string, value any) error {
	return s.store.FlashMessage(w, r, key, value)
}
func (s *captureStore) TakeMessages(w http.ResponseWriter, r *http.Request) (map[string]any, error) {
	flash, err := s.store.TakeMessages(w, r)
	messagesFrom(r).flash = flash
	return flash, err
}
func (s *captureStore) FlushResponse(w http.ResponseWriter) error { return s.store.FlushResponse(w) }
