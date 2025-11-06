package resource

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/AD7six/dd-tf/internal/config"
)

type fakeHTTPClient struct {
	resp *http.Response
	err  error
}

func (f *fakeHTTPClient) Get(url string) (*http.Response, error) {
	return f.resp, f.err
}

func TestFetchResourceFromAPI_HappyPath(t *testing.T) {
	body := `{"foo":"bar","n":123}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
	settings := &config.Settings{HTTPMaxBodySize: 1024}
	client := &fakeHTTPClient{resp: resp}

	got, err := FetchResourceFromAPI(client, "https://api.example.com/v1/x", settings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["foo"].(string) != "bar" {
		t.Fatalf("expected foo=bar, got %v", got["foo"])
	}
}

func TestFetchResourceFromAPI_Non200(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Status:     "500 Internal Server Error",
		Body:       io.NopCloser(bytes.NewBufferString("oops")),
	}
	settings := &config.Settings{HTTPMaxBodySize: 1024}
	client := &fakeHTTPClient{resp: resp}

	_, err := FetchResourceFromAPI(client, "https://api.example.com/v1/x", settings)
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}
