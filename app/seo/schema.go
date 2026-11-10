package seo

// JSON-LD builders (https://schema.org). Plain maps keep the output explicit.

type Obj = map[string]any

func Org(name, url string) Obj {
	return Obj{"@type": "Organization", "@id": url + "/#organization", "name": name, "url": url}
}

func WebSite(name, url, searchURLTemplate, lang string) Obj {
	return Obj{
		"@context": "https://schema.org", "@type": "WebSite", "@id": url + "/#website", "name": name, "url": url, "inLanguage": lang,
		"publisher":       Obj{"@id": url + "/#organization"},
		"potentialAction": Obj{"@type": "SearchAction", "target": Obj{"@type": "EntryPoint", "urlTemplate": searchURLTemplate}, "query-input": "required name=search_term_string"},
	}
}

type Crumb struct{ Name, URL string }

func Breadcrumbs(crumbs []Crumb) Obj {
	items := make([]Obj, 0, len(crumbs))
	for i, c := range crumbs {
		items = append(items, Obj{"@type": "ListItem", "position": i + 1, "name": c.Name, "item": c.URL})
	}
	return Obj{"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": items}
}

type ListItem struct{ Name, URL string }

func CollectionPage(name, description, url, lang, siteURL string, items []ListItem) Obj {
	list := make([]Obj, 0, len(items))
	for i, it := range items {
		list = append(list, Obj{"@type": "ListItem", "position": i + 1, "name": it.Name, "url": it.URL})
	}
	return Obj{
		"@context": "https://schema.org", "@type": "CollectionPage", "name": name, "description": description, "url": url, "inLanguage": lang,
		"isPartOf":   Obj{"@id": siteURL + "/#website"},
		"mainEntity": Obj{"@type": "ItemList", "numberOfItems": len(items), "itemListElement": list},
	}
}

type SoftwareInput struct {
	Name, Description, URL, Lang string
	Category, Language, License  string
	RepoURL, Homepage, Version   string
	Image                        string
	Screenshots                  []string
	Topics                       []string
	PublishedISO, ModifiedISO    string
	Video                        *VideoInput
}

type VideoInput struct {
	Name, Description, EmbedURL, ContentURL, Thumbnail, UploadDate string
}

// Software describes the project as a SoftwareApplication (free, open source).
func Software(in SoftwareInput) Obj {
	o := Obj{
		"@context": "https://schema.org", "@type": "SoftwareApplication", "name": in.Name, "description": in.Description,
		"url": in.URL, "inLanguage": in.Lang, "applicationCategory": "DeveloperApplication", "operatingSystem": "Cross-platform",
		"isAccessibleForFree": true, "offers": Obj{"@type": "Offer", "price": "0", "priceCurrency": "USD"},
	}
	if in.Category != "" {
		o["applicationSubCategory"] = in.Category
	}
	if in.Language != "" {
		o["programmingLanguage"] = in.Language
	}
	if in.License != "" {
		o["license"] = in.License
	}
	if in.RepoURL != "" {
		o["codeRepository"] = in.RepoURL
		o["sameAs"] = []string{in.RepoURL}
	}
	if in.Homepage != "" {
		o["installUrl"] = in.Homepage
	}
	if in.Version != "" {
		o["softwareVersion"] = in.Version
	}
	if in.Image != "" {
		o["image"] = in.Image
	}
	if len(in.Screenshots) > 0 {
		o["screenshot"] = in.Screenshots
	}
	if len(in.Topics) > 0 {
		o["keywords"] = in.Topics
	}
	if in.PublishedISO != "" {
		o["dateCreated"] = in.PublishedISO
	}
	if in.ModifiedISO != "" {
		o["dateModified"] = in.ModifiedISO
	}
	return o
}

func Video(in VideoInput) Obj {
	return Obj{
		"@context": "https://schema.org", "@type": "VideoObject", "name": in.Name, "description": in.Description,
		"thumbnailUrl": []string{in.Thumbnail}, "uploadDate": in.UploadDate, "embedUrl": in.EmbedURL, "contentUrl": in.ContentURL,
	}
}
