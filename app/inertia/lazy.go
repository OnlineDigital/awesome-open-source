// Package inertia provides explicitly application-owned additions to inertia-go.
// Native Optional, Defer, Once and merge builders remain upstream types.
package inertia

import (
	"fmt"
	"net/http"
	"strings"

	upstream "github.com/inertia-go/inertia-go"
)

type lazyProp struct{ fn func() (any, error) }

// Lazy computes an ordinary prop only if selected for this response. Unlike
// Optional it is included on a full visit. This does not depend on Merge bugs.
func Lazy(fn func() (any, error)) any { return lazyProp{fn: fn} }

// Render resolves only our Lazy wrappers, then delegates protocol and native
// builder metadata to inertia-go. version must match the adapter configuration.
func Render(adapter *upstream.Inertia, version string, w http.ResponseWriter, r *http.Request, component string, props upstream.Props) {
	info := upstream.FromRequest(r)
	// Let upstream version negotiation run before any application callback.
	if info.IsInertia && r.Method == http.MethodGet && info.Version != "" && info.Version != version {
		adapter.Render(w, r, component, props)
		return
	}
	partial := info.IsInertia && info.PartialComponent == component
	resolved, err := resolveMap(map[string]any(props), "", partial, info.PartialData, info.PartialExcept)
	if err != nil {
		http.Error(w, "Failed to load page data", http.StatusInternalServerError)
		return
	}
	adapter.Render(w, r, component, upstream.Props(resolved))
}

func selected(path string, partial bool, only, except []string) bool {
	if !partial {
		return true
	}
	for _, selector := range except {
		if path == selector || strings.HasPrefix(path, selector+".") {
			return false
		}
	}
	if len(only) == 0 {
		return true
	}
	for _, selector := range only {
		if path == selector || strings.HasPrefix(path, selector+".") || strings.HasPrefix(selector, path+".") {
			return true
		}
	}
	return false
}

func resolveMap(props map[string]any, prefix string, partial bool, only, except []string) (map[string]any, error) {
	result := make(map[string]any, len(props))
	for key, value := range props {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}
		resolved, include, err := resolveValue(value, path, partial, only, except)
		if err != nil {
			return nil, err
		}
		if include {
			result[key] = resolved
		}
	}
	return result, nil
}

func resolveValue(value any, path string, partial bool, only, except []string) (any, bool, error) {
	switch v := value.(type) {
	case lazyProp:
		if !selected(path, partial, only, except) {
			return nil, false, nil
		}
		if v.fn == nil {
			return nil, false, fmt.Errorf("nil callback for %s", path)
		}
		result, err := v.fn()
		if err != nil {
			return nil, false, fmt.Errorf("load %s: %w", path, err)
		}
		return resolveValue(result, path, partial, only, except)
	case map[string]any:
		resolved, err := resolveMap(v, path, partial, only, except)
		return resolved, true, err
	case []any:
		// Arrays inherit their prop path: partial selectors address map fields,
		// not numeric indexes, matching the native array resolver.
		result := make([]any, 0, len(v))
		for _, item := range v {
			resolved, include, err := resolveValue(item, path, partial, only, except)
			if err != nil {
				return nil, false, err
			}
			if include {
				result = append(result, resolved)
			}
		}
		return result, true, nil
	default:
		// Keep native builders intact: upstream owns their evaluation and filters.
		return value, true, nil
	}
}
