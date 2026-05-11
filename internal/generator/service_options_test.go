package generator

import (
	"strings"
	"testing"
)

func TestParseServiceRPCCGOOptions(t *testing.T) {
	tests := []struct {
		name     string
		comments string
		want     []AdapterToken
	}{
		{
			name:     "defaults to msg connect when annotation is absent",
			comments: "Service comment without generator options.",
			want:     []AdapterToken{AdapterTokenMessageConnect},
		},
		{
			name:     "parses msg connect",
			comments: "@rpccgo:msg-connect",
			want:     []AdapterToken{AdapterTokenMessageConnect},
		},
		{
			name:     "parses msg grpc",
			comments: "@rpccgo:msg-grpc",
			want:     []AdapterToken{AdapterTokenMessageGRPC},
		},
		{
			name:     "parses both message adapters",
			comments: "@rpccgo:msg-connect|msg-grpc",
			want:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC},
		},
		{
			name:     "parses message connect plus native",
			comments: "@rpccgo:msg-connect|native",
			want:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative},
		},
		{
			name:     "parses message grpc plus native",
			comments: "@rpccgo:msg-grpc|native",
			want:     []AdapterToken{AdapterTokenMessageGRPC, AdapterTokenNative},
		},
		{
			name:     "parses all adapters",
			comments: "@rpccgo:msg-connect|msg-grpc|native",
			want:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC, AdapterTokenNative},
		},
		{
			name:     "expands native only to msg connect plus native",
			comments: "@rpccgo:native",
			want:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative},
		},
		{
			name:     "deduplicates and canonicalizes adapter tokens",
			comments: "@rpccgo:native|msg-grpc|msg-connect|native|msg-grpc",
			want:     []AdapterToken{AdapterTokenMessageConnect, AdapterTokenMessageGRPC, AdapterTokenNative},
		},
		{
			name: "finds annotation in service leading comments",
			comments: `// Greeter serves greeting requests.
// @rpccgo: msg-connect | native
// More service docs.`,
			want: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative},
		},
		{
			name: "ignores inline mention in prose",
			comments: `// Greeter docs mention @rpccgo:msg-grpc as an example.
// No actual directive here.`,
			want: []AdapterToken{AdapterTokenMessageConnect},
		},
		{
			name: "treats native and expanded native directives as equivalent",
			comments: `// @rpccgo:native
// @rpccgo:msg-connect|native`,
			want: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseServiceRPCCGOOptions(tt.comments)
			if err != nil {
				t.Fatalf("ParseServiceRPCCGOOptions() error = %v", err)
			}
			assertAdapterTokens(t, got, tt.want)
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

func TestAdapterSelectionHas(t *testing.T) {
	selection := AdapterSelection{Tokens: []AdapterToken{AdapterTokenMessageConnect, AdapterTokenNative}}

	if !selection.Has(AdapterTokenMessageConnect) {
		t.Fatalf("selection.Has(%q) = false, want true", AdapterTokenMessageConnect)
	}
	if !selection.Has(AdapterTokenNative) {
		t.Fatalf("selection.Has(%q) = false, want true", AdapterTokenNative)
	}
	if selection.Has(AdapterTokenMessageGRPC) {
		t.Fatalf("selection.Has(%q) = true, want false", AdapterTokenMessageGRPC)
	}
}

func assertAdapterTokens(t *testing.T, got AdapterSelection, want []AdapterToken) {
	t.Helper()

	if len(got.Tokens) != len(want) {
		t.Fatalf("tokens = %#v, want %#v", got.Tokens, want)
	}
	for i := range want {
		if got.Tokens[i] != want[i] {
			t.Fatalf("tokens = %#v, want %#v", got.Tokens, want)
		}
	}
}
