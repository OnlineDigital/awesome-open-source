package protocol_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	inertia "github.com/inertia-go/inertia-go"
)

func nativeHeaders(only string) map[string]string {
	h := map[string]string{"X-Inertia": "true", "X-Inertia-Version": "native-test"}
	if only != "" {
		h["X-Inertia-Partial-Component"] = "Probe"
		h["X-Inertia-Partial-Data"] = only
	}
	return h
}

func nativeJSON(t *testing.T, h http.Handler, headers map[string]string) map[string]any {
	t.Helper()
	w := request(t, h, "GET", "/probe", "", headers)
	requireEqual(t, w.Code, 200)
	var p map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNativeOnceAliasExpiryFreshAndOptional(t *testing.T) {
	i := nativeAdapter(t)
	calls := 0
	fresh := false
	h := nativeRender(i, func() inertia.Props {
		v := inertia.Once(func() (any, error) {
			calls++
			if fresh {
				return nil, nil
			}
			return "signed-in", nil
		}).As("identity").ExpiresIn(time.Minute)
		if fresh {
			v.Fresh()
		}
		return inertia.Props{"user": v, "optional": inertia.Optional(func() (any, error) { return "value", nil }).Once().As("opt")}
	})
	start := time.Now().UnixMilli()
	p := nativeJSON(t, h, nativeHeaders(""))
	requireEqual(t, nativeProp(p, "onceProps.identity.prop"), "user")
	exp, ok := nativeProp(p, "onceProps.identity.expiresAt").(float64)
	if !ok || exp < float64(start+59000) || exp > float64(start+61000) {
		t.Fatalf("invalid expiry timestamp: %v", exp)
	}
	requireEqual(t, nativeProp(p, "props.optional"), nil)
	requireEqual(t, nativeProp(p, "onceProps.opt.prop"), "optional")
	cache := nativeHeaders("")
	cache["X-Inertia-Except-Once-Props"] = "identity"
	nativeJSON(t, h, cache)
	requireEqual(t, calls, 1)
	p = nativeJSON(t, h, nativeHeaders("optional"))
	requireEqual(t, nativeProp(p, "props.optional"), "value")
	fresh = true
	p = nativeJSON(t, h, cache)
	user, present := p["props"].(map[string]any)["user"]
	if !present || user != nil {
		t.Fatalf("Fresh must include explicit null after logout simulation: %v", p)
	}
	requireEqual(t, calls, 2)
}

func TestNativeDeferredCompositionRescueAndRetry(t *testing.T) {
	i := nativeAdapter(t)
	calls := 0
	h := nativeRender(i, func() inertia.Props {
		return inertia.Props{
			"a": inertia.Defer(func() (any, error) { return []int{1}, nil }, "team").DeepMerge().Once(),
			"b": inertia.Defer(func() (any, error) {
				calls++
				if calls == 1 {
					return nil, errors.New("controlled failure")
				}
				return "recovered", nil
			}, "team").Rescue(),
		}
	})
	p := nativeJSON(t, h, nativeHeaders(""))
	requireEqual(t, p["deferredProps"], map[string]any{"team": []any{"a", "b"}})
	requireEqual(t, p["deepMergeProps"], []any{"a"})
	requireEqual(t, nativeProp(p, "onceProps.a.prop"), "a")
	requireEqual(t, calls, 0)
	p = nativeJSON(t, h, nativeHeaders("a,b"))
	requireEqual(t, p["rescuedProps"], []any{"b"})
	requireEqual(t, nativeProp(p, "props.a"), []any{float64(1)})
	requireEqual(t, nativeProp(p, "props.b"), nil)
	p = nativeJSON(t, h, nativeHeaders("b"))
	requireEqual(t, nativeProp(p, "props.b"), "recovered")
	requireEqual(t, p["rescuedProps"], nil)
}

func TestNativePrependNestedMatchAndReset(t *testing.T) {
	i := nativeAdapter(t)
	h := nativeRender(i, func() inertia.Props {
		return inertia.Props{
			"root":   inertia.Merge([]int{1}).Prepend(),
			"feed":   inertia.Merge(map[string]any{"older": []int{1}, "newer": []int{2}}).Prepend("older").Append("newer").MatchOn(map[string]string{"older": "id", "newer": "id"}),
			"cached": inertia.Merge(func() (any, error) { return []int{3}, nil }).Once(),
		}
	})
	p := nativeJSON(t, h, nativeHeaders("root,feed,cached"))
	requireEqual(t, p["prependProps"], []any{"feed.older", "root"})
	requireEqual(t, p["mergeProps"], []any{"cached", "feed.newer"})
	requireEqual(t, p["matchPropsOn"], []any{"feed.newer.id", "feed.older.id"})
	requireEqual(t, nativeProp(p, "onceProps.cached.prop"), "cached")
	headers := nativeHeaders("root,feed,cached")
	headers["X-Inertia-Reset"] = "root,feed,cached"
	p = nativeJSON(t, h, headers)
	requireEqual(t, p["mergeProps"], nil)
	requireEqual(t, p["prependProps"], nil)
	requireEqual(t, p["matchPropsOn"], nil)
}

func TestKnownUpstreamSessionAndValidationLimits(t *testing.T) {
	t.Run("flash lives in props and named bags flatten", func(t *testing.T) {
		i := nativeAdapter(t)
		h := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/save" {
				inertia.Flash(r).Set("message", "saved")
				inertia.ValidationErrors(r).Bag("profile").Add("name", "required")
				i.Redirect(w, r, "/probe")
				return
			}
			i.Render(w, r, "Probe", inertia.Props{})
		}))
		w := request(t, h, "POST", "/save", "", nativeHeaders(""))
		cookies := w.Result().Cookies()
		headers := nativeHeaders("")
		headers["X-Inertia-Error-Bag"] = "profile"
		w = request(t, h, "GET", "/probe", "", headers, cookies...)
		var p map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		requireEqual(t, nativeProp(p, "props.flash.message"), "saved")
		requireEqual(t, p["flash"], nil)
		requireEqual(t, nativeProp(p, "props.errors.name"), "required")
		requireEqual(t, nativeProp(p, "props.errors.profile"), nil)
	})
	t.Run("stale409 consumes flash and errors", func(t *testing.T) {
		i := nativeAdapter(t)
		h := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/save" {
				inertia.Flash(r).Set("message", "saved")
				inertia.ValidationErrors(r).Add("name", "required")
				i.Redirect(w, r, "/probe")
				return
			}
			i.Render(w, r, "Probe", inertia.Props{})
		}))
		w := request(t, h, "POST", "/save", "", nativeHeaders(""))
		cookies := w.Result().Cookies()
		headers := nativeHeaders("")
		headers["X-Inertia-Version"] = "old"
		w = request(t, h, "GET", "/probe", "", headers, cookies...)
		requireEqual(t, w.Code, 409)
		w = request(t, h, "GET", "/probe", "", nativeHeaders(""), cookies...)
		var p map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		requireEqual(t, nativeProp(p, "props.flash"), nil)
		requireEqual(t, nativeProp(p, "props.errors"), map[string]any{})
	})
	t.Run("native Precognition422 has strings and no message", func(t *testing.T) {
		i := nativeAdapter(t)
		h := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			i.HandlePrecognition(w, r, func(r *http.Request) { inertia.ValidationErrors(r).Add("name", "required") })
		}))
		w := request(t, h, "POST", "/validate", "", map[string]string{"Precognition": "true"})
		requireEqual(t, w.Code, 422)
		var p map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		requireEqual(t, nativeProp(p, "errors.name"), "required")
		requireEqual(t, p["message"], nil)
	})
	t.Run("native Scroll omits paginator siblings", func(t *testing.T) {
		i := nativeAdapter(t)
		next := 2
		h := nativeRender(i, func() inertia.Props {
			return inertia.Props{"items": inertia.Scroll(inertia.ScrollConfig{CurrentPage: 1, PageName: "page", NextPage: &next}, func() any { return []int{1} })}
		})
		p := nativeJSON(t, h, nativeHeaders(""))
		requireEqual(t, nativeProp(p, "props.items.data"), []any{float64(1)})
		requireEqual(t, nativeProp(p, "props.items.current_page"), nil)
		requireEqual(t, nativeProp(p, "props.items.next_page_url"), nil)
		requireEqual(t, nativeProp(p, "scrollProps.items.nextPage"), float64(2))
	})
}
