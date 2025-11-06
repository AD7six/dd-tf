package templating

import (
	"reflect"
	"testing"
)

func TestExtractTagMap(t *testing.T) {
	tests := []struct {
		name     string
		raw      any
		sanitize bool
		want     map[string]string
	}{
		{
			name:     "nil input",
			raw:      nil,
			sanitize: false,
			want:     map[string]string{},
		},
		{
			name:     "empty slice",
			raw:      []any{},
			sanitize: false,
			want:     map[string]string{},
		},
		{
			name: "simple tags",
			raw: []any{
				"team:platform",
				"env:prod",
				"service:api",
			},
			sanitize: false,
			want: map[string]string{
				"team":    "platform",
				"env":     "prod",
				"service": "api",
			},
		},
		{
			name: "tags with spaces",
			raw: []any{
				"team: platform ",
				" env : prod",
				"service:  api  ",
			},
			sanitize: false,
			want: map[string]string{
				"team":    "platform",
				"env":     "prod",
				"service": "api",
			},
		},
		{
			name: "tags without colons ignored",
			raw: []any{
				"team:platform",
				"invalid-tag",
				"env:prod",
			},
			sanitize: false,
			want: map[string]string{
				"team": "platform",
				"env":  "prod",
			},
		},
		{
			name: "tags with multiple colons",
			raw: []any{
				"url:https://example.com:8080",
				"team:platform",
			},
			sanitize: false,
			want: map[string]string{
				"url":  "https://example.com:8080",
				"team": "platform",
			},
		},
		{
			name: "sanitize enabled",
			raw: []any{
				"team:platform Team",
				"service:My/Service",
			},
			sanitize: true,
			want: map[string]string{
				"team":    "platform-Team",
				"service": "My-Service",
			},
		},
		{
			name: "sanitize disabled",
			raw: []any{
				"team:platform Team",
				"service:My/Service",
			},
			sanitize: false,
			want: map[string]string{
				"team":    "platform Team",
				"service": "My/Service",
			},
		},
		{
			name: "non-string items ignored",
			raw: []any{
				"team:platform",
				123,
				true,
				"env:prod",
			},
			sanitize: false,
			want: map[string]string{
				"team": "platform",
				"env":  "prod",
			},
		},
		{
			name:     "wrong type input",
			raw:      "not-a-slice",
			sanitize: false,
			want:     map[string]string{},
		},
		{
			name: "duplicate keys - last wins",
			raw: []any{
				"team:frontend",
				"team:platform",
			},
			sanitize: false,
			want: map[string]string{
				"team": "platform",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTagMap(tt.raw, tt.sanitize)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractTagMap(%v, %v) = %v, want %v", tt.raw, tt.sanitize, got, tt.want)
			}
		})
	}
}

func TestHasAllTagsMap(t *testing.T) {
	tests := []struct {
		name       string
		tags       map[string]string
		filterTags []string
		want       bool
	}{
		{
			name:       "empty filter always matches",
			tags:       map[string]string{"team": "platform"},
			filterTags: []string{},
			want:       true,
		},
		{
			name:       "nil filter always matches",
			tags:       map[string]string{"team": "platform"},
			filterTags: nil,
			want:       true,
		},
		{
			name: "exact match",
			tags: map[string]string{
				"team": "platform",
				"env":  "prod",
			},
			filterTags: []string{"team:platform"},
			want:       true,
		},
		{
			name: "all filters match",
			tags: map[string]string{
				"team":    "platform",
				"env":     "prod",
				"service": "api",
			},
			filterTags: []string{"team:platform", "env:prod"},
			want:       true,
		},
		{
			name: "one filter missing",
			tags: map[string]string{
				"team": "platform",
				"env":  "prod",
			},
			filterTags: []string{"team:platform", "service:api"},
			want:       false,
		},
		{
			name:       "empty tags, non-empty filter",
			tags:       map[string]string{},
			filterTags: []string{"team:platform"},
			want:       false,
		},
		{
			name: "case insensitive match",
			tags: map[string]string{
				"team": "platform",
				"env":  "PROD",
			},
			filterTags: []string{"TEAM:platform", "ENV:prod"},
			want:       true,
		},
		{
			name: "case insensitive key and value",
			tags: map[string]string{
				"Team": "platform",
			},
			filterTags: []string{"team:platform"},
			want:       true,
		},
		{
			name: "partial match not enough",
			tags: map[string]string{
				"team": "platform",
			},
			filterTags: []string{"team:platform", "team:frontend"},
			want:       false,
		},
		{
			name: "multiple filters all present",
			tags: map[string]string{
				"team":     "platform",
				"env":      "prod",
				"service":  "api",
				"priority": "1",
			},
			filterTags: []string{"team:platform", "env:prod", "priority:1"},
			want:       true,
		},
		{
			name: "wrong value",
			tags: map[string]string{
				"team": "platform",
			},
			filterTags: []string{"team:frontend"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasAllTagsMap(tt.tags, tt.filterTags)
			if got != tt.want {
				t.Errorf("HasAllTagsMap(%v, %v) = %v, want %v", tt.tags, tt.filterTags, got, tt.want)
			}
		})
	}
}

func TestHasAllTagsSlice(t *testing.T) {
	tests := []struct {
		name          string
		dashboardTags []string
		filterTags    []string
		want          bool
	}{
		{
			name:          "empty filter always matches",
			dashboardTags: []string{"team:platform"},
			filterTags:    []string{},
			want:          true,
		},
		{
			name:          "nil filter always matches",
			dashboardTags: []string{"team:platform"},
			filterTags:    nil,
			want:          true,
		},
		{
			name:          "exact match",
			dashboardTags: []string{"team:platform", "env:prod"},
			filterTags:    []string{"team:platform"},
			want:          true,
		},
		{
			name:          "all filters present",
			dashboardTags: []string{"team:platform", "env:prod", "service:api"},
			filterTags:    []string{"team:platform", "env:prod"},
			want:          true,
		},
		{
			name:          "one filter missing",
			dashboardTags: []string{"team:platform", "env:prod"},
			filterTags:    []string{"team:platform", "service:api"},
			want:          false,
		},
		{
			name:          "empty dashboard tags, non-empty filter",
			dashboardTags: []string{},
			filterTags:    []string{"team:platform"},
			want:          false,
		},
		{
			name:          "nil dashboard tags, non-empty filter",
			dashboardTags: nil,
			filterTags:    []string{"team:platform"},
			want:          false,
		},
		{
			name:          "case insensitive match",
			dashboardTags: []string{"Team:platform", "ENV:PROD"},
			filterTags:    []string{"team:platform", "env:prod"},
			want:          true,
		},
		{
			name:          "mixed case",
			dashboardTags: []string{"TEAM:platform", "env:PROD"},
			filterTags:    []string{"team:platform", "ENV:prod"},
			want:          true,
		},
		{
			name:          "duplicate tags in dashboard",
			dashboardTags: []string{"team:platform", "team:platform", "env:prod"},
			filterTags:    []string{"team:platform"},
			want:          true,
		},
		{
			name:          "all filters match with extras",
			dashboardTags: []string{"team:platform", "env:prod", "service:api", "priority:1"},
			filterTags:    []string{"team:platform", "priority:1"},
			want:          true,
		},
		{
			name:          "partial tag value match fails",
			dashboardTags: []string{"team:platform"},
			filterTags:    []string{"team:back"},
			want:          false,
		},
		{
			name:          "substring not a match",
			dashboardTags: []string{"team:platform-service"},
			filterTags:    []string{"team:platform"},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasAllTagsSlice(tt.dashboardTags, tt.filterTags)
			if got != tt.want {
				t.Errorf("HasAllTagsSlice(%v, %v) = %v, want %v", tt.dashboardTags, tt.filterTags, got, tt.want)
			}
		})
	}
}
