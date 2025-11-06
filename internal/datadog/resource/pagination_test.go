package resource

import "testing"

func TestOffsetPagination(t *testing.T) {
	p := NewOffsetPagination(100)
	if p.Start != 0 || p.Count != 100 {
		t.Fatalf("unexpected initial state: start=%d count=%d", p.Start, p.Count)
	}
	url := p.FormatOffsetURL("https://api.example.com/v1/items")
	if url != "https://api.example.com/v1/items?start=0&count=100" {
		t.Fatalf("unexpected url: %s", url)
	}
	if !p.NextOffsetPage(100) {
		t.Fatalf("expected more pages when itemsReceived equals count")
	}
	if p.Start != 100 {
		t.Fatalf("expected start to advance to 100, got %d", p.Start)
	}
	if p.NextOffsetPage(10) {
		t.Fatalf("expected no more pages when itemsReceived < count")
	}
}

func TestPagePagination(t *testing.T) {
	p := NewPagePagination(50)
	if p.Page != 0 || p.PageSize != 50 {
		t.Fatalf("unexpected initial state: page=%d pageSize=%d", p.Page, p.PageSize)
	}
	url := p.FormatPageURL("https://api.example.com/v1/monitors")
	if url != "https://api.example.com/v1/monitors?page=0&page_size=50" {
		t.Fatalf("unexpected url: %s", url)
	}
	if !p.NextPage(50) {
		t.Fatalf("expected more pages when itemsReceived equals page size")
	}
	if p.Page != 1 {
		t.Fatalf("expected page to advance to 1, got %d", p.Page)
	}
	if p.NextPage(10) {
		t.Fatalf("expected no more pages when itemsReceived < page size")
	}
}
