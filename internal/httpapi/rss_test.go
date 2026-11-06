package httpapi

import "testing"

func TestParseFeed(t *testing.T) {
	rss := `<?xml version="1.0"?>
<rss version="2.0"><channel>
<title>Example feed</title><link>https://example.com/</link>
<atom:link href="https://example.com/feed.xml" rel="self" xmlns:atom="http://www.w3.org/2005/Atom"/>
<description>Notes</description>
<item><title>First</title><link>https://example.com/1</link><guid>1</guid><pubDate>Mon, 03 Nov 2025 10:00:00 GMT</pubDate></item>
<item><title>Second</title><link>https://example.com/2</link></item>
</channel></rss>`
	feed, err := parseFeed([]byte(rss))
	if err != nil {
		t.Fatalf("rss: %v", err)
	}
	// The atom:link self reference must not overwrite the channel link: both
	// match the xml:"link" field by local name.
	if feed.Title != "Example feed" || feed.Link != "https://example.com/" || len(feed.Items) != 2 {
		t.Fatalf("rss feed = %+v", feed)
	}
	if feed.Items[0].Title != "First" || feed.Items[0].Link != "https://example.com/1" ||
		feed.Items[0].GUID != "1" || feed.Items[0].PubDate == "" {
		t.Errorf("first item = %+v", feed.Items[0])
	}

	atom := `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
<title>Atom feed</title>
<link rel="alternate" href="https://atom.example/"/>
<entry><title>Entry one</title><id>urn:1</id><published>2025-11-03T10:00:00Z</published>
<link rel="alternate" href="https://atom.example/1"/></entry>
<entry><title>Entry two</title><id>urn:2</id><updated>2025-11-04T10:00:00Z</updated>
<link href="https://atom.example/2"/></entry>
</feed>`
	feed, err = parseFeed([]byte(atom))
	if err != nil {
		t.Fatalf("atom: %v", err)
	}
	if feed.Title != "Atom feed" || feed.Link != "https://atom.example/" || len(feed.Items) != 2 {
		t.Fatalf("atom feed = %+v", feed)
	}
	if feed.Items[0].Link != "https://atom.example/1" || feed.Items[0].PubDate == "" {
		t.Errorf("atom first item = %+v", feed.Items[0])
	}
	// An entry without published falls back to updated.
	if feed.Items[1].PubDate == "" {
		t.Errorf("atom second item has no date")
	}

	if _, err := parseFeed([]byte(`<html><body>not a feed</body></html>`)); err == nil {
		t.Error("html accepted as a feed")
	}
	if _, err := parseFeed([]byte(`not xml`)); err == nil {
		t.Error("plain text accepted as a feed")
	}

	empty, err := parseFeed([]byte(`<rss version="2.0"><channel><title>Empty</title></channel></rss>`))
	if err != nil {
		t.Fatalf("empty rss: %v", err)
	}
	if empty.Items == nil || len(empty.Items) != 0 {
		t.Errorf("empty feed items = %v, want an empty array", empty.Items)
	}
}
