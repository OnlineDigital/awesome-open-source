package inertia

import (
	"errors"
	"testing"
)

func TestLazyNestedSelectorsAvoidUnwantedCallbacks(t *testing.T) {
	for _, tt := range []struct {
		name                                     string
		partial                                  bool
		only, except                             []string
		wantUsers, wantNotifications, wantSecret int
	}{
		{"full", false, nil, nil, 1, 1, 1},
		{"only users", true, []string{"users"}, nil, 1, 0, 0},
		{"only descendant", true, []string{"auth.notifications"}, nil, 0, 1, 0},
		{"except parent", true, nil, []string{"auth"}, 1, 0, 0},
		{"except child", true, nil, []string{"auth.secret"}, 1, 1, 0},
		{"except wins", true, []string{"auth"}, []string{"auth.notifications"}, 0, 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			users, notifications, secret := 0, 0, 0
			props := map[string]any{"users": Lazy(func() (any, error) { users++; return "users", nil }), "auth": map[string]any{"notifications": Lazy(func() (any, error) { notifications++; return "notifications", nil }), "secret": Lazy(func() (any, error) { secret++; return "secret", nil })}}
			_, err := resolveMap(props, "", tt.partial, tt.only, tt.except)
			if err != nil {
				t.Fatal(err)
			}
			if users != tt.wantUsers || notifications != tt.wantNotifications || secret != tt.wantSecret {
				t.Fatalf("callback counts %d,%d,%d want %d,%d,%d", users, notifications, secret, tt.wantUsers, tt.wantNotifications, tt.wantSecret)
			}
		})
	}
}

func TestLazyErrorsAndExcludedFailure(t *testing.T) {
	props := map[string]any{"bad": Lazy(func() (any, error) { return nil, errors.New("database failed") })}
	if _, err := resolveMap(props, "", false, nil, nil); err == nil {
		t.Fatal("callback error was ignored")
	}
	if result, err := resolveMap(props, "", true, []string{"other"}, nil); err != nil || len(result) != 0 {
		t.Fatalf("excluded callback was evaluated: %v %v", result, err)
	}
}

func TestNilCallbackFailsOnlyWhenSelected(t *testing.T) {
	props := map[string]any{"bad": Lazy(nil)}
	if _, err := resolveMap(props, "", false, nil, nil); err == nil {
		t.Fatal("nil callback should fail when selected")
	}
	if result, err := resolveMap(props, "", true, nil, []string{"bad"}); err != nil || len(result) != 0 {
		t.Fatalf("excluded nil callback should not run: %v %v", result, err)
	}
}
