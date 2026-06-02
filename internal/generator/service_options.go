package generator

import (
	"fmt"
	"strings"
)

const serviceRPCCGODirective = "@rpccgo:"

type serviceGenerationToken string

const (
	serviceGenerationTokenMessageConnect serviceGenerationToken = "msg-connect"
	serviceGenerationTokenMessageGRPC    serviceGenerationToken = "msg-grpc"
	serviceGenerationTokenNative         serviceGenerationToken = "native"
)

var canonicalServiceGenerationTokens = []serviceGenerationToken{
	serviceGenerationTokenMessageConnect,
	serviceGenerationTokenMessageGRPC,
	serviceGenerationTokenNative,
}

// ParseServiceRPCCGOOptions parses the service leading comment text for the
// @rpccgo directive. Descriptor-specific extraction is intentionally left to
// the descriptor planning layer so this parser can be tested as a pure string
// contract.
func ParseServiceRPCCGOOptions(comments string) (ServiceGenerationSelection, error) {
	directives := serviceRPCCGODirectives(comments)
	if len(directives) == 0 {
		return ServiceGenerationSelection{MessageTransport: MessageTransportConnect}, nil
	}

	var selection ServiceGenerationSelection
	var canonical string
	var firstDirective string
	for i, directive := range directives {
		parsed, err := parseServiceRPCCGODirective(directive)
		if err != nil {
			return ServiceGenerationSelection{}, err
		}

		current := serviceGenerationSelectionKey(parsed)
		if i == 0 {
			selection = parsed
			canonical = current
			firstDirective = strings.TrimSpace(directive)
			continue
		}
		if current != canonical {
			return ServiceGenerationSelection{}, fmt.Errorf("conflicting @rpccgo directives: %q selects %q but %q selects %q",
				firstDirective, canonical, strings.TrimSpace(directive), current)
		}
	}

	return selection, nil
}

func parseServiceRPCCGODirective(directive string) (ServiceGenerationSelection, error) {
	if strings.Contains(directive, ":") {
		return ServiceGenerationSelection{}, fmt.Errorf("invalid @rpccgo directive %q: repeated ':' is not allowed", directive)
	}

	trimmed := strings.TrimSpace(directive)
	if trimmed == "" {
		return ServiceGenerationSelection{}, fmt.Errorf("empty @rpccgo directive")
	}

	seen := make(map[serviceGenerationToken]bool)
	for _, rawToken := range strings.Split(trimmed, "|") {
		token := strings.TrimSpace(rawToken)
		if token == "" {
			return ServiceGenerationSelection{}, fmt.Errorf("empty @rpccgo token in directive %q", directive)
		}

		parsedToken := serviceGenerationToken(token)
		if !isKnownServiceGenerationToken(parsedToken) {
			return ServiceGenerationSelection{}, fmt.Errorf("unknown @rpccgo token %q; valid tokens: msg-connect, msg-grpc, native", token)
		}
		seen[parsedToken] = true
	}

	if len(seen) == 1 && seen[serviceGenerationTokenNative] {
		seen[serviceGenerationTokenMessageConnect] = true
	}
	if seen[serviceGenerationTokenMessageConnect] && seen[serviceGenerationTokenMessageGRPC] {
		return ServiceGenerationSelection{}, fmt.Errorf("@rpccgo message transport must select exactly one of msg-connect or msg-grpc")
	}

	return serviceGenerationSelectionFromSet(seen), nil
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

func serviceGenerationSelectionFromSet(seen map[serviceGenerationToken]bool) ServiceGenerationSelection {
	selection := ServiceGenerationSelection{
		NativeEnabled: seen[serviceGenerationTokenNative],
	}
	switch {
	case seen[serviceGenerationTokenMessageGRPC]:
		selection.MessageTransport = MessageTransportGRPC
	default:
		selection.MessageTransport = MessageTransportConnect
	}
	return selection
}

func serviceGenerationSelectionKey(selection ServiceGenerationSelection) string {
	parts := []string{}
	switch selection.MessageTransport {
	case MessageTransportConnect:
		parts = append(parts, string(serviceGenerationTokenMessageConnect))
	case MessageTransportGRPC:
		parts = append(parts, string(serviceGenerationTokenMessageGRPC))
	}
	if selection.NativeEnabled {
		parts = append(parts, string(serviceGenerationTokenNative))
	}
	return strings.Join(parts, "|")
}

func isKnownServiceGenerationToken(token serviceGenerationToken) bool {
	for _, known := range canonicalServiceGenerationTokens {
		if token == known {
			return true
		}
	}
	return false
}
