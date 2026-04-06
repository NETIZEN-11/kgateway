package httproute

import (
	"path"
	"regexp"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/kgateway/query"
	delegationutils "github.com/kgateway-dev/kgateway/v2/pkg/kgateway/utils/delegation"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
)

// filterDelegatedChildren takes a parent route matcher and a list of children
// referenced by the parent's backendRefs, and filters the children based on
// the following criteria, returning only the valid child delegatee routes:
//   - If a child sets parentRefs, the parentRefs must include the parent (if
//     parentRefs is not set, then any parent may delegate to that child)
//   - If route matcher inheritance is used (via annotation on the child), the
//     child matcher does not need to match the parent matcher. The parent and
//     child matchers are merged according to the rules specified by
//     `mergeParentChildRouteMatch`.
//   - If route matcher inheritance is not used (the default), then the parent
//     and child matchers must match according to the requirements specified by
//     `isDelegatedRouteMatch`. If they don't match, the child matcher will be
//     discarded from the results.
//
// After the above processing, if a child route rule does not have any valid
// matches with respect to the parent, the rule is discarded. If the child route
// does not have any remaining valid route rules, the whole route is discarded.
func filterDelegatedChildren(
	parentRef types.NamespacedName,
	parentMatch gwv1.HTTPRouteMatch,
	children []*query.RouteInfo,
) []*query.RouteInfo {
	// Select the child routes that match the parent
	var selected []*query.RouteInfo
	for _, c := range children {
		// Check if the child route is allowed to be delegated to by the parent
		if !delegationutils.ChildRouteCanAttachToParentRef(c.Object.GetNamespace(), c.Object.GetParentRefs(), parentRef) {
			continue
		}

		// make a copy; multiple parents can delegate to the same child so we can't modify a shared reference
		clone := c.Clone()
		origChild, ok := clone.Object.(*ir.HttpRouteIR)
		if !ok {
			continue
		}
		cloneChild := *origChild
		child := &cloneChild
		// make sure we don't overwite the original rules
		child.Rules = make([]ir.HttpRouteRuleIR, len(origChild.Rules))
		copy(child.Rules, origChild.Rules)

		inheritMatcher := child.DelegationInheritParentMatcher

		// We use validRules to store the rules in the child route that are valid
		// (matches in the rule match the parent route matcher). If a specific rule
		// in the child is not valid, then we discard it in the final child route
		// returned by this function.
		var validRules []ir.HttpRouteRuleIR
		for i, rule := range child.Rules {
			// We use validMatches to store the matches in the child rule that are valid
			// with respect to the parent matcher.
			var validMatches []gwv1.HTTPRouteMatch

			// If the child route opts to inherit the parent's matcher and it does not specify its own matcher,
			// simply inherit the parent's matcher.
			if inheritMatcher && len(rule.Matches) == 0 {
				validMatches = append(validMatches, parentMatch)
			}

			for _, match := range rule.Matches {
				match := *match.DeepCopy()
				if inheritMatcher {
					// When inheriting the parent's matcher, all matches are valid.
					// In this case, the child inherits the parents matcher so we merge
					// the parent's matcher with the child's.
					mergeParentChildRouteMatch(&parentMatch, &match)
					validMatches = append(validMatches, match)
				} else if ok := delegationutils.IsDelegatedRouteMatch(parentMatch, match); ok {
					// Non-inherited matcher delegation requires matching child matcher to parent matcher
					// to delegate from the parent route to the child.
					validMatches = append(validMatches, match)
				}
			}

			// if there were any valid matches, store this rule as a valid rule
			if len(validMatches) > 0 {
				validRule := child.Rules[i]
				validRule.Matches = validMatches
				validRules = append(validRules, validRule)
			}
		}
		// if there were any valid rules, then add this child route as a valid delegatee
		if len(validRules) > 0 {
			child.Rules = validRules
			clone.Object = child
			selected = append(selected, clone)
		}
	}

	return selected
}

// mergeParentChildRouteMatch is called only when inherit-parent-matcher is set.
// It merges the parent route match into the child as follows:
//   - the resulting path consists of parent path + child path
//   - the resulting headers consist of the combined headers from parent and child, with parent header taking
//     precedence on any name conflicts
//   - the resulting query parameters consist of the combined query parameters from parent and child, with parent
//     query params taking precedence on any name conflicts
//   - the child inherits the parent's method if specified; otherwise the child retains its own method
func mergeParentChildRouteMatch(
	parent *gwv1.HTTPRouteMatch,
	child *gwv1.HTTPRouteMatch,
) {
	if parent == nil || child == nil {
		return
	}

	if child.Path == nil {
		child.Path = &gwv1.HTTPPathMatch{
			Type:  ptr.To(gwv1.PathMatchPathPrefix),
			Value: new(""),
		}
	}

	// Extract parent path information with nil checks
	parentPathType := gwv1.PathMatchPathPrefix
	parentPathValue := ""
	if parent.Path != nil {
		parentPathType = ptr.Deref(parent.Path.Type, gwv1.PathMatchPathPrefix)
		if parent.Path.Value != nil {
			parentPathValue = *parent.Path.Value
		}
	}

	// Extract child path information with nil checks
	childPathType := ptr.Deref(child.Path.Type, gwv1.PathMatchPathPrefix)
	childPathValue := ""
	if child.Path.Value != nil {
		childPathValue = *child.Path.Value
	}

	// Merge paths based on parent path type
	if parentPathType == gwv1.PathMatchRegularExpression {
		// Regex-aware path merging
		child.Path.Value = ptr.To(mergeRegexPath(parentPathValue, childPathValue, childPathType))
		child.Path.Type = ptr.To(gwv1.PathMatchRegularExpression)
	} else {
		// Standard path joining for PathPrefix and Exact types
		child.Path.Value = ptr.To(path.Join(parentPathValue, childPathValue))
		child.Path.Type = ptr.To(childPathType)
	}

	// Inherit parent and child headers and query parameters while augmenting the merge
	// with additions specified on the child
	child.Headers = mergeHeaders(parent.Headers, child.Headers)
	child.QueryParams = mergeQueries(parent.QueryParams, child.QueryParams)

	// If parent specifies a method, inherit it (this will overwrite any method specified on the child)
	if parent.Method != nil {
		child.Method = new(*parent.Method)
	}
}

// mergeRegexPath merges a parent regex path with a child path, handling regex-specific logic.
// It removes trailing anchors from the parent, escapes the child path if needed, and
// reconstructs a valid regex pattern.
func mergeRegexPath(parentRegex, childPath string, childPathType gwv1.PathMatchType) string {
	// Remove trailing $ anchor from parent regex if present
	trimmedParent := strings.TrimSuffix(parentRegex, "$")

	// Strip trailing optional path groups and $ anchor before merging
	// Matches patterns like (?:/...)?, (?:...)?, (/.*)?, etc.
	trailingOptionalGroup := regexp.MustCompile(`\(\?:?[^)]*\)\??$`)
	trimmedParent = trailingOptionalGroup.ReplaceAllString(trimmedParent, "")

	// Escape child path based on its type
	var escapedChild string
	if childPathType == gwv1.PathMatchRegularExpression {
		// Child is already a regex, wrap in non-capturing group to preserve semantics
		childRegex := strings.TrimPrefix(childPath, "^")
		childRegex = strings.TrimSuffix(childRegex, "$")
		escapedChild = "(?:" + childRegex + ")"
	} else {
		// Child is PathPrefix or Exact, escape special regex characters
		escapedChild = escapeRegexSpecialChars(childPath)
	}

	// Ensure child path starts with / if not empty and not already wrapped in group
	if escapedChild != "" && !strings.HasPrefix(escapedChild, "/") && !strings.HasPrefix(escapedChild, "(?:") {
		escapedChild = "/" + escapedChild
	}

	// If child path is empty, return parent regex as-is (preserving the $ anchor)
	if escapedChild == "" {
		return parentRegex
	}

	// Construct the merged regex based on child path type
	var mergedRegex string
	if childPathType == gwv1.PathMatchPathPrefix {
		// PathPrefix: allow additional path segments
		mergedRegex = trimmedParent + escapedChild + "(/.*)?$"
	} else if childPathType == gwv1.PathMatchExact {
		// Exact: match exactly
		mergedRegex = trimmedParent + escapedChild + "$"
	} else {
		// RegularExpression: child already wrapped, no extra suffix needed
		mergedRegex = trimmedParent + escapedChild + "$"
	}

	return mergedRegex
}

// escapeRegexSpecialChars escapes special regex characters in a plain string path
func escapeRegexSpecialChars(s string) string {
	return regexp.QuoteMeta(s)
}

// mergeHeaders merges parent and child header matches. If a header name is specified on both
// the parent and child, the parent's header value takes precedence (i.e. child cannot overwrite it).
func mergeHeaders(
	parent, child []gwv1.HTTPHeaderMatch,
) []gwv1.HTTPHeaderMatch {
	merged := make(map[gwv1.HTTPHeaderName]gwv1.HTTPHeaderMatch)
	for _, h := range parent {
		merged[h.Name] = h
	}
	for _, h := range child {
		key := h.Name
		// Only add the child if it does not conflict with the parent
		if _, ok := merged[key]; !ok {
			merged[key] = h
		}
	}
	var result []gwv1.HTTPHeaderMatch
	for _, h := range merged {
		result = append(result, h)
	}
	// Sort for deterministic ordering
	slices.SortFunc(result, func(a, b gwv1.HTTPHeaderMatch) int {
		return strings.Compare(string(a.Name), string(b.Name))
	})
	return result
}

// mergeQueries merges parent and child query param matches. If a query param name is specified on both
// the parent and child, the parent's query param value takes precedence (i.e. child cannot overwrite it).
func mergeQueries(
	parent, child []gwv1.HTTPQueryParamMatch,
) []gwv1.HTTPQueryParamMatch {
	merged := make(map[gwv1.HTTPHeaderName]gwv1.HTTPQueryParamMatch)
	for _, h := range parent {
		merged[h.Name] = h
	}
	for _, h := range child {
		key := h.Name
		// Only add the child if it does not conflict with the parent
		if _, ok := merged[key]; !ok {
			merged[key] = h
		}
	}
	var result []gwv1.HTTPQueryParamMatch
	for _, h := range merged {
		result = append(result, h)
	}
	// Sort for deterministic ordering
	slices.SortFunc(result, func(a, b gwv1.HTTPQueryParamMatch) int {
		return strings.Compare(string(a.Name), string(b.Name))
	})
	return result
}
