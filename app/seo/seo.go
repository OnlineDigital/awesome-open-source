// Package seo builds <head> tags, schema.org JSON-LD, sitemaps and robots.txt.
package seo

import (
	"encoding/json"
	"html"
	"html/template"
	"sort"
	"strings"
)

// Tag is one element to place in <head>. The server renders it into the first
// response; the Solid app re-creates the same tags on client-side navigation.
type Tag struct {
	Kind    string            `json:"kind"` // meta | link | script
	Attrs   map[string]string `json:"attrs"`
	Content string            `json:"content,omitempty"` // script body (JSON-LD)
}

type Alternate struct {
	Locale string `json:"locale"` // BCP47 code or "x-default"
	URL    string `json:"url"`
}

type Meta struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Canonical   string      `json:"canonical"`
	Image       string      `json:"image,omitempty"`
	Robots      string      `json:"robots,omitempty"`
	Locale      string      `json:"locale"`
	OGLocale    string      `json:"-"`
	OGType      string      `json:"-"`
	SiteName    string      `json:"-"`
	Alternates  []Alternate `json:"alternates"`
	Markdown    string      `json:"markdown,omitempty"` // absolute URL of the .md/.txt twin
	JSONLD      []any       `json:"-"`
	Tags        []Tag       `json:"tags"`
	// Fallback is plain server-rendered content for crawlers/agents that do not run JavaScript.
	Fallback template.HTML `json:"-"`
}

// Build fills Tags from the other fields. Call after all fields are set.
func (m *Meta) Build() *Meta {
	meta := func(k, v string, val string) {
		if val != "" {
			m.Tags = append(m.Tags, Tag{Kind: "meta", Attrs: map[string]string{k: v, "content": val}})
		}
	}
	link := func(attrs map[string]string) { m.Tags = append(m.Tags, Tag{Kind: "link", Attrs: attrs}) }
	m.Tags = nil
	meta("name", "description", m.Description)
	robots := m.Robots
	if robots == "" {
		robots = "index,follow,max-image-preview:large,max-snippet:-1,max-video-preview:-1"
	}
	meta("name", "robots", robots)
	if m.Canonical != "" {
		link(map[string]string{"rel": "canonical", "href": m.Canonical})
	}
	for _, a := range m.Alternates {
		link(map[string]string{"rel": "alternate", "hreflang": a.Locale, "href": a.URL})
	}
	if m.Markdown != "" {
		link(map[string]string{"rel": "alternate", "type": "text/markdown", "href": m.Markdown})
	}
	ogType := m.OGType
	if ogType == "" {
		ogType = "website"
	}
	meta("property", "og:type", ogType)
	meta("property", "og:site_name", m.SiteName)
	meta("property", "og:title", m.Title)
	meta("property", "og:description", m.Description)
	meta("property", "og:url", m.Canonical)
	meta("property", "og:locale", m.OGLocale)
	meta("property", "og:image", m.Image)
	card := "summary"
	if m.Image != "" {
		card = "summary_large_image"
	}
	meta("name", "twitter:card", card)
	meta("name", "twitter:title", m.Title)
	meta("name", "twitter:description", m.Description)
	meta("name", "twitter:image", m.Image)
	for _, ld := range m.JSONLD {
		b, err := json.Marshal(ld)
		if err != nil {
			continue
		}
		m.Tags = append(m.Tags, Tag{Kind: "script", Attrs: map[string]string{"type": "application/ld+json"}, Content: string(b)})
	}
	return m
}

// HeadHTML renders <title> and all tags. Every element carries data-seo so the client can replace them.
func (m *Meta) HeadHTML() template.HTML {
	var b strings.Builder
	b.WriteString(`<title data-seo>` + html.EscapeString(m.Title) + `</title>` + "\n")
	for _, t := range m.Tags {
		keys := make([]string, 0, len(t.Attrs))
		for k := range t.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		switch t.Kind {
		case "script":
			b.WriteString(`<script data-seo`)
		default:
			b.WriteString(`<` + t.Kind + ` data-seo`)
		}
		for _, k := range keys {
			b.WriteString(` ` + k + `="` + html.EscapeString(t.Attrs[k]) + `"`)
		}
		if t.Kind == "script" {
			// JSON-LD must not be able to close the script element.
			safe := strings.NewReplacer("<", `<`, ">", `>`, "&", `&`, " ", ` `, " ", ` `).Replace(t.Content)
			b.WriteString(`>` + safe + `</script>`)
		} else {
			b.WriteString(`>`)
		}
		b.WriteByte('\n')
	}
	return template.HTML(b.String())
}

type Link struct{ Text, URL, Note string }
type Section struct {
	Heading string
	Links   []Link
}

// Fallback renders plain semantic HTML for the <noscript> block (crawlers and agents without JavaScript).
func Fallback(h1, intro string, sections []Section) template.HTML {
	var b strings.Builder
	b.WriteString("<main><h1>" + html.EscapeString(h1) + "</h1>")
	if intro != "" {
		b.WriteString("<p>" + html.EscapeString(intro) + "</p>")
	}
	for _, s := range sections {
		if len(s.Links) == 0 {
			continue
		}
		if s.Heading != "" {
			b.WriteString("<h2>" + html.EscapeString(s.Heading) + "</h2>")
		}
		b.WriteString("<ul>")
		for _, l := range s.Links {
			b.WriteString(`<li><a href="` + html.EscapeString(l.URL) + `">` + html.EscapeString(l.Text) + "</a>")
			if l.Note != "" {
				b.WriteString(" — " + html.EscapeString(l.Note))
			}
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
	}
	b.WriteString("</main>")
	return template.HTML(b.String())
}

// Truncate shortens s to at most n runes, appending an ellipsis.
func Truncate(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) <= n {
		return string(r)
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}
