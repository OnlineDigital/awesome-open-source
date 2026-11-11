package seo

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

type SitemapURL struct {
	Loc        string
	LastMod    time.Time
	Alternates []Alternate
	Images     []string
	Video      *SitemapVideo
}

type SitemapVideo struct {
	Thumbnail, Title, Description, Player, PublishedISO string
}

func esc(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// URLSet renders a sitemap with xhtml:link hreflang alternates, image and video extensions.
func URLSet(urls []SitemapURL) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1" xmlns:video="http://www.google.com/schemas/sitemap-video/1.1">` + "\n")
	for _, u := range urls {
		b.WriteString("<url><loc>" + esc(u.Loc) + "</loc>")
		if !u.LastMod.IsZero() {
			b.WriteString("<lastmod>" + u.LastMod.UTC().Format("2006-01-02") + "</lastmod>")
		}
		for _, a := range u.Alternates {
			fmt.Fprintf(&b, `<xhtml:link rel="alternate" hreflang="%s" href="%s"/>`, esc(a.Locale), esc(a.URL))
		}
		for _, img := range u.Images {
			b.WriteString("<image:image><image:loc>" + esc(img) + "</image:loc></image:image>")
		}
		if v := u.Video; v != nil {
			b.WriteString("<video:video><video:thumbnail_loc>" + esc(v.Thumbnail) + "</video:thumbnail_loc><video:title>" + esc(v.Title) + "</video:title><video:description>" + esc(v.Description) + "</video:description><video:player_loc>" + esc(v.Player) + "</video:player_loc>")
			if v.PublishedISO != "" {
				b.WriteString("<video:publication_date>" + esc(v.PublishedISO) + "</video:publication_date>")
			}
			b.WriteString("</video:video>")
		}
		b.WriteString("</url>\n")
	}
	b.WriteString("</urlset>\n")
	return []byte(b.String())
}

type SitemapRef struct {
	Loc     string
	LastMod time.Time
}

func Index(refs []SitemapRef) []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, r := range refs {
		b.WriteString("<sitemap><loc>" + esc(r.Loc) + "</loc>")
		if !r.LastMod.IsZero() {
			b.WriteString("<lastmod>" + r.LastMod.UTC().Format("2006-01-02") + "</lastmod>")
		}
		b.WriteString("</sitemap>\n")
	}
	b.WriteString("</sitemapindex>\n")
	return []byte(b.String())
}

// Robots explicitly welcomes search engines and AI crawlers; admin and API stay private.
func Robots(baseURL string) string {
	bots := []string{"GPTBot", "OAI-SearchBot", "ChatGPT-User", "ClaudeBot", "Claude-User", "Claude-SearchBot", "PerplexityBot", "Google-Extended", "Applebot-Extended", "CCBot"}
	var b strings.Builder
	b.WriteString("User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /api/\nDisallow: /mcp/\n\n")
	for _, bot := range bots {
		b.WriteString("User-agent: " + bot + "\nAllow: /\nDisallow: /admin\n\n")
	}
	b.WriteString("Sitemap: " + baseURL + "/sitemap.xml\n")
	b.WriteString("# LLM-friendly index: " + baseURL + "/llms.txt\n")
	return b.String()
}
