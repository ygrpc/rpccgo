package generator

import (
	"strings"
	"testing"
)

func TestParseServiceRPCCGOOptions(t *testing.T) {
	tests := []struct {
		name     string
		comments string
		want     ServiceGenerationSelection
	}{
		{
			name:     "defaults to msg connect when annotation is absent",
			comments: "Service comment without generator options.",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportConnect},
		},
		{
			name:     "parses msg connect",
			comments: "@rpccgo:msg-connect",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportConnect},
		},
		{
			name:     "parses msg grpc",
			comments: "@rpccgo:msg-grpc",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportGRPC},
		},
		{
			name:     "parses message connect plus native",
			comments: "@rpccgo:msg-connect|native",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
		},
		{
			name:     "parses message grpc plus native",
			comments: "@rpccgo:msg-grpc|native",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportGRPC, NativeEnabled: true},
		},
		{
			name:     "expands native only to msg connect plus native",
			comments: "@rpccgo:native",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
		},
		{
			name:     "deduplicates and canonicalizes adapter tokens",
			comments: "@rpccgo:native|msg-grpc|native|msg-grpc",
			want:     ServiceGenerationSelection{MessageTransport: MessageTransportGRPC, NativeEnabled: true},
		},
		{
			name: "finds annotation in service leading comments",
			comments: `// Greeter serves greeting requests.
// @rpccgo: msg-connect | native
// More service docs.`,
			want: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
		},
		{
			name: "ignores inline mention in prose",
			comments: `// Greeter docs mention @rpccgo:msg-grpc as an example.
// No actual directive here.`,
			want: ServiceGenerationSelection{MessageTransport: MessageTransportConnect},
		},
		{
			name: "treats native and expanded native directives as equivalent",
			comments: `// @rpccgo:native
// @rpccgo:msg-connect|native`,
			want: ServiceGenerationSelection{MessageTransport: MessageTransportConnect, NativeEnabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceRPCCGOOptions(tt.comments)
			if err != nil {
				t.Fatalf("ParseServiceRPCCGOOptions() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseServiceRPCCGOOptions() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseServiceRPCCGOOptionsErrors(t *testing.T) {
	tests := []struct {
		name        string
		comments    string
		wantMessage string
	}{
		{
			name:        "unknown token includes legal token hint",
			comments:    "@rpccgo:msg-conenct",
			wantMessage: "valid tokens: msg-connect, msg-grpc, native",
		},
		{
			name:        "unknown token keeps bad token in error",
			comments:    "@rpccgo:msg-connect|bogus",
			wantMessage: `unknown @rpccgo token "bogus"`,
		},
		{
			name:        "spelling error keeps bad token in error",
			comments:    "@rpccgo:msg-conenct",
			wantMessage: `unknown @rpccgo token "msg-conenct"`,
		},
		{
			name:        "empty directive is rejected",
			comments:    "@rpccgo:",
			wantMessage: "empty @rpccgo directive",
		},
		{
			name:        "blank token is rejected",
			comments:    "@rpccgo:msg-connect| |native",
			wantMessage: "empty @rpccgo token",
		},
		{
			name:        "repeated colon is rejected",
			comments:    "@rpccgo:msg-connect:msg-grpc",
			wantMessage: "invalid @rpccgo directive",
		},
		{
			name:        "simultaneous message transports are rejected",
			comments:    "@rpccgo:msg-connect|msg-grpc",
			wantMessage: "@rpccgo message transport must select exactly one of msg-connect or msg-grpc",
		},
		{
			name:        "simultaneous message transports with native are rejected",
			comments:    "@rpccgo:msg-connect|msg-grpc|native",
			wantMessage: "@rpccgo message transport must select exactly one of msg-connect or msg-grpc",
		},
		{
			name:        "conflicting repeated directives are rejected",
			comments:    "@rpccgo:msg-connect\n@rpccgo:msg-grpc",
			wantMessage: `conflicting @rpccgo directives: "msg-connect" selects "msg-connect" but "msg-grpc" selects "msg-grpc"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseServiceRPCCGOOptions(tt.comments)
			if err == nil {
				t.Fatalf("ParseServiceRPCCGOOptions() error = nil, want message containing %q", tt.wantMessage)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("ParseServiceRPCCGOOptions() error = %q, want message containing %q", err.Error(), tt.wantMessage)
			}
		})
	}
}

func TestServiceGenerationSelectionHasIdentity(t *testing.T) {
	tests := []struct {
		name      string
		selection ServiceGenerationSelection
		want      bool
	}{
		{
			name:      "connect transport is initialized",
			selection: ServiceGenerationSelection{MessageTransport: MessageTransportConnect},
			want:      true,
		},
		{
			name:      "grpc transport is initialized",
			selection: ServiceGenerationSelection{MessageTransport: MessageTransportGRPC},
			want:      true,
		},
		{
			name:      "zero transport is uninitialized",
			selection: ServiceGenerationSelection{},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.selection.HasIdentity(); got != tt.want {
				t.Fatalf("HasIdentity() = %v, want %v", got, tt.want)
			}
		})
	}
}
