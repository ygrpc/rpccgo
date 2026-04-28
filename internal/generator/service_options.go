package generator

import (
	"fmt"
	"strings"
)

const serviceRPCCGODirective = "@rpccgo:"

var canonicalAdapterTokens = []AdapterToken{
	AdapterTokenMessageConnect,
	AdapterTokenMessageGRPC,
	AdapterTokenNative,
}

// ParseServiceRPCCGOOptions parses the service leading comment text for the
// @rpccgo directive. Descriptor-specific extraction is intentionally left to
// the descriptor planning layer so this parser can be tested as a pure string
// contract.
func ParseServiceRPCCGOOptions(comments string) (AdapterSelection, error) {
	directives := serviceRPCCGODirectives(comments)
	if len(directives) == 0 {
		return AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect}}, nil
	}

	var selection AdapterSelection
	var canonical string
	var firstDirective string
	for i, directive := range directives {
		parsed, err := parseServiceRPCCGODirective(directive)
		if err != nil {
			return AdapterSelection{}, err
		}

		current := adapterSelectionKey(parsed)
		if i == 0 {
			selection = parsed
			canonical = current
			firstDirective = strings.TrimSpace(directive)
			continue
		}
		if current != canonical {
			return AdapterSelection{}, fmt.Errorf("conflicting @rpccgo directives: %q selects %q but %q selects %q",
				firstDirective, canonical, strings.TrimSpace(directive), current)
		}
	}

	return selection, nil
}

func parseServiceRPCCGODirective(directive string) (AdapterSelection, error) {
	if strings.Contains(directive, ":") {
		return AdapterSelection{}, fmt.Errorf("invalid @rpccgo directive %q: repeated ':' is not allowed", directive)
	}

	trimmed := strings.TrimSpace(directive)
	if trimmed == "" {
		return AdapterSelection{}, fmt.Errorf("empty @rpccgo directive")
	}

	seen := make(map[AdapterToken]bool)
	for _, rawToken := range strings.Split(trimmed, "|") {
		token := strings.TrimSpace(rawToken)
		if token == "" {
			return AdapterSelection{}, fmt.Errorf("empty @rpccgo token in directive %q", directive)
		}

		adapterToken := AdapterToken(token)
		if !isKnownAdapterToken(adapterToken) {
			return AdapterSelection{}, fmt.Errorf("unknown @rpccgo token %q; valid tokens: msg-connect, msg-grpc, native", token)
		}
		seen[adapterToken] = true
	}

	if len(seen) == 1 && seen[AdapterTokenNative] {
		seen[AdapterTokenMessageConnect] = true
	}

	return adapterSelectionFromSet(seen), nil
}

func serviceRPCCGODirectives(comments string) []string {
	var directives []string
	for _, line := range strings.Split(comments, "\n") {
		trimmed := strings.TrimSpace(line)
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "//"))
		if strings.HasPrefix(trimmed, serviceRPCCGODirective) {
			directives = append(directives, trimmed[len(serviceRPCCGODirective):])
		}
	}
	return directives
}

func adapterSelectionFromSet(seen map[AdapterToken]bool) AdapterSelection {
	tokens := make([]AdapterToken, 0, len(canonicalAdapterTokens))
	for _, token := range canonicalAdapterTokens {
		if seen[token] {
			tokens = append(tokens, token)
		}
	}
	return AdapterSelection{Tokens: tokens}
}

func adapterSelectionKey(selection AdapterSelection) string {
	parts := make([]string, 0, len(selection.Tokens))
	for _, token := range selection.Tokens {
		parts = append(parts, string(token))
	}
	return strings.Join(parts, "|")
}

func isKnownAdapterToken(token AdapterToken) bool {
	for _, known := range canonicalAdapterTokens {
		if token == known {
			return true
		}
	}
	return false
}
