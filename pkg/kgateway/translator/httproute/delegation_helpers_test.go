package httproute

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestMergeParentChildRouteMatch(t *testing.T) {
	tests := []struct {
		name           string
		parent         *gwv1.HTTPRouteMatch
		child          *gwv1.HTTPRouteMatch
		expectedPath   string
		expectedType   gwv1.PathMatchType
		expectedMethod *gwv1.HTTPMethod
	}{
		{
			name: "PathPrefix parent + PathPrefix child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/v1"),
				},
			},
			expectedPath: "/api/v1",
			expectedType: gwv1.PathMatchPathPrefix,
		},
		{
			name: "PathPrefix parent + Exact child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchExact),
					Value: ptr.To("/users"),
				},
			},
			expectedPath: "/api/users",
			expectedType: gwv1.PathMatchExact,
		},
		{
			name: "RegularExpression parent + PathPrefix child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/teams/[^/]+$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/members"),
				},
			},
			expectedPath: "^/teams/[^/]+/members$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "RegularExpression parent with optional group + PathPrefix child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/teams/[^/]+(?:/.*)?$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/members"),
				},
			},
			expectedPath: "^/teams/[^/]+/members$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "RegularExpression parent + Exact child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api/v[0-9]+$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchExact),
					Value: ptr.To("/users"),
				},
			},
			expectedPath: "^/api/v[0-9]+/users$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "RegularExpression parent + RegularExpression child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/teams/[^/]+$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("/members/[0-9]+"),
				},
			},
			expectedPath: "^/teams/[^/]+/members/[0-9]+$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "RegularExpression parent without anchor + PathPrefix child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/teams/[^/]+"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/members"),
				},
			},
			expectedPath: "^/teams/[^/]+/members$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Child with special regex characters in path",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/v1.0"),
				},
			},
			expectedPath: "^/api/v1\\.0$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Empty child path with PathPrefix parent",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To(""),
				},
			},
			expectedPath: "/api",
			expectedType: gwv1.PathMatchPathPrefix,
		},
		{
			name: "Empty child path with RegularExpression parent",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api/[^/]+$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To(""),
				},
			},
			expectedPath: "^/api/[^/]+$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Child path without leading slash",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("users"),
				},
			},
			expectedPath: "^/api/users$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Nil child path",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: nil,
			},
			expectedPath: "/api",
			expectedType: gwv1.PathMatchPathPrefix,
		},
		{
			name: "Method inheritance from parent",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
				Method: ptr.To(gwv1.HTTPMethodGet),
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/users"),
				},
				Method: ptr.To(gwv1.HTTPMethodPost),
			},
			expectedPath:   "/api/users",
			expectedType:   gwv1.PathMatchPathPrefix,
			expectedMethod: ptr.To(gwv1.HTTPMethodGet),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make copies to avoid modifying test data
			parent := tt.parent.DeepCopy()
			child := tt.child.DeepCopy()

			mergeParentChildRouteMatch(parent, child)

			assert.NotNil(t, child.Path)
			assert.NotNil(t, child.Path.Value)
			assert.Equal(t, tt.expectedPath, *child.Path.Value, "path value mismatch")
			assert.NotNil(t, child.Path.Type)
			assert.Equal(t, tt.expectedType, *child.Path.Type, "path type mismatch")

			if tt.expectedMethod != nil {
				assert.NotNil(t, child.Method)
				assert.Equal(t, *tt.expectedMethod, *child.Method, "method mismatch")
			}
		})
	}
}

func TestMergeParentChildRouteMatch_HeadersAndQueryParams(t *testing.T) {
	tests := []struct {
		name                  string
		parent                *gwv1.HTTPRouteMatch
		child                 *gwv1.HTTPRouteMatch
		expectedHeaderCount   int
		expectedQueryCount    int
		expectedHeaderNames   []string
		expectedQueryNames    []string
	}{
		{
			name: "Merge headers - parent takes precedence",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
				Headers: []gwv1.HTTPHeaderMatch{
					{
						Name:  "X-Custom-Header",
						Value: "parent-value",
					},
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/users"),
				},
				Headers: []gwv1.HTTPHeaderMatch{
					{
						Name:  "X-Custom-Header",
						Value: "child-value",
					},
					{
						Name:  "X-Child-Header",
						Value: "child-only",
					},
				},
			},
			expectedHeaderCount: 2,
			expectedHeaderNames: []string{"X-Custom-Header", "X-Child-Header"},
		},
		{
			name: "Merge query params - parent takes precedence",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api"),
				},
				QueryParams: []gwv1.HTTPQueryParamMatch{
					{
						Name:  "version",
						Value: "v1",
					},
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/users"),
				},
				QueryParams: []gwv1.HTTPQueryParamMatch{
					{
						Name:  "version",
						Value: "v2",
					},
					{
						Name:  "limit",
						Value: "10",
					},
				},
			},
			expectedQueryCount: 2,
			expectedQueryNames: []string{"limit", "version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := tt.parent.DeepCopy()
			child := tt.child.DeepCopy()

			mergeParentChildRouteMatch(parent, child)

			if tt.expectedHeaderCount > 0 {
				assert.Len(t, child.Headers, tt.expectedHeaderCount)
				headerNames := make([]string, len(child.Headers))
				for i, h := range child.Headers {
					headerNames[i] = string(h.Name)
				}
				assert.ElementsMatch(t, tt.expectedHeaderNames, headerNames)
			}

			if tt.expectedQueryCount > 0 {
				assert.Len(t, child.QueryParams, tt.expectedQueryCount)
				queryNames := make([]string, len(child.QueryParams))
				for i, q := range child.QueryParams {
					queryNames[i] = string(q.Name)
				}
				assert.ElementsMatch(t, tt.expectedQueryNames, queryNames)
			}
		})
	}
}

func TestMergeRegexPath(t *testing.T) {
	tests := []struct {
		name          string
		parentRegex   string
		childPath     string
		childPathType gwv1.PathMatchType
		expected      string
	}{
		{
			name:          "Basic regex with anchor",
			parentRegex:   "^/teams/[^/]+$",
			childPath:     "/members",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/teams/[^/]+/members$",
		},
		{
			name:          "Regex with optional trailing group",
			parentRegex:   "^/teams/[^/]+(?:/.*)?$",
			childPath:     "/members",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/teams/[^/]+/members$",
		},
		{
			name:          "Regex without trailing anchor",
			parentRegex:   "^/api/v[0-9]+",
			childPath:     "/users",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/api/v[0-9]+/users$",
		},
		{
			name:          "Child path with special characters",
			parentRegex:   "^/api$",
			childPath:     "/v1.0+test",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/api/v1\\.0\\+test$",
		},
		{
			name:          "Child is also regex",
			parentRegex:   "^/teams/[^/]+$",
			childPath:     "/members/[0-9]+",
			childPathType: gwv1.PathMatchRegularExpression,
			expected:      "^/teams/[^/]+/members/[0-9]+$",
		},
		{
			name:          "Child regex with both anchors",
			parentRegex:   "^/api$",
			childPath:     "^/v[0-9]+/users$",
			childPathType: gwv1.PathMatchRegularExpression,
			expected:      "^/api/v[0-9]+/users$",
		},
		{
			name:          "Empty child path",
			parentRegex:   "^/api/[^/]+$",
			childPath:     "",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/api/[^/]+$",
		},
		{
			name:          "Child path without leading slash",
			parentRegex:   "^/api$",
			childPath:     "users",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/api/users$",
		},
		{
			name:          "Complex regex pattern",
			parentRegex:   "^/org/[a-z0-9-]+/projects/[^/]+$",
			childPath:     "/issues",
			childPathType: gwv1.PathMatchPathPrefix,
			expected:      "^/org/[a-z0-9-]+/projects/[^/]+/issues$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeRegexPath(tt.parentRegex, tt.childPath, tt.childPathType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeRegexSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No special characters",
			input:    "/api/users",
			expected: "/api/users",
		},
		{
			name:     "Dot character",
			input:    "/v1.0",
			expected: "/v1\\.0",
		},
		{
			name:     "Plus character",
			input:    "/test+path",
			expected: "/test\\+path",
		},
		{
			name:     "Multiple special characters",
			input:    "/api/v1.0+test?query",
			expected: "/api/v1\\.0\\+test\\?query",
		},
		{
			name:     "Brackets and parentheses",
			input:    "/path[0]/test(1)",
			expected: "/path\\[0\\]/test\\(1\\)",
		},
		{
			name:     "All special characters",
			input:    ".+*?^$()[]{}|\\",
			expected: "\\.\\+\\*\\?\\^\\$\\(\\)\\[\\]\\{\\}\\|\\\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeRegexSpecialChars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMergeParentChildRouteMatch_NilCases(t *testing.T) {
	t.Run("Nil parent", func(t *testing.T) {
		child := &gwv1.HTTPRouteMatch{
			Path: &gwv1.HTTPPathMatch{
				Type:  ptr.To(gwv1.PathMatchPathPrefix),
				Value: ptr.To("/users"),
			},
		}
		originalChild := child.DeepCopy()

		mergeParentChildRouteMatch(nil, child)

		// Child should remain unchanged
		assert.Equal(t, originalChild, child)
	})

	t.Run("Nil child", func(t *testing.T) {
		parent := &gwv1.HTTPRouteMatch{
			Path: &gwv1.HTTPPathMatch{
				Type:  ptr.To(gwv1.PathMatchPathPrefix),
				Value: ptr.To("/api"),
			},
		}

		mergeParentChildRouteMatch(parent, nil)
		// Should not panic
	})

	t.Run("Both nil", func(t *testing.T) {
		mergeParentChildRouteMatch(nil, nil)
		// Should not panic
	})
}

func TestMergeParentChildRouteMatch_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		parent       *gwv1.HTTPRouteMatch
		child        *gwv1.HTTPRouteMatch
		expectedPath string
		expectedType gwv1.PathMatchType
	}{
		{
			name: "Regex with multiple optional groups",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api/v[0-9]+(?:/.*)?$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/users"),
				},
			},
			expectedPath: "^/api/v[0-9]+/users$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Regex with complex pattern and special chars in child",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/org/[a-zA-Z0-9_-]+$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/projects/v2.0"),
				},
			},
			expectedPath: "^/org/[a-zA-Z0-9_-]+/projects/v2\\.0$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Regex parent with child regex containing anchors",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/v[0-9]+/users$"),
				},
			},
			expectedPath: "^/api/v[0-9]+/users$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "PathPrefix with trailing slash",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/api/"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/users"),
				},
			},
			expectedPath: "/api/users",
			expectedType: gwv1.PathMatchPathPrefix,
		},
		{
			name: "Regex with nested groups",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/teams/(public|private)/[^/]+$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/members"),
				},
			},
			expectedPath: "^/teams/(public|private)/[^/]+/members$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
		{
			name: "Child path with backslash (Windows-style path)",
			parent: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchRegularExpression),
					Value: ptr.To("^/api$"),
				},
			},
			child: &gwv1.HTTPRouteMatch{
				Path: &gwv1.HTTPPathMatch{
					Type:  ptr.To(gwv1.PathMatchPathPrefix),
					Value: ptr.To("/path\\with\\backslash"),
				},
			},
			expectedPath: "^/api/path\\\\with\\\\backslash$",
			expectedType: gwv1.PathMatchRegularExpression,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := tt.parent.DeepCopy()
			child := tt.child.DeepCopy()

			mergeParentChildRouteMatch(parent, child)

			assert.NotNil(t, child.Path)
			assert.NotNil(t, child.Path.Value)
			assert.Equal(t, tt.expectedPath, *child.Path.Value, "path value mismatch")
			assert.NotNil(t, child.Path.Type)
			assert.Equal(t, tt.expectedType, *child.Path.Type, "path type mismatch")
		})
	}
}
