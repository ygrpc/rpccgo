package generator

import "testing"

func TestStreamingKindOf(t *testing.T) {
	tests := []struct {
		name           string
		isClientStream bool
		isServerStream bool
		want           StreamingKind
	}{
		{name: "unary", want: StreamingKindUnary},
		{name: "client streaming", isClientStream: true, want: StreamingKindClientStreaming},
		{name: "server streaming", isServerStream: true, want: StreamingKindServerStreaming},
		{name: "bidi streaming", isClientStream: true, isServerStream: true, want: StreamingKindBidiStreaming},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StreamingKindOf(tt.isClientStream, tt.isServerStream)
			if got != tt.want {
				t.Fatalf("StreamingKindOf(%v, %v) = %v, want %v", tt.isClientStream, tt.isServerStream, got, tt.want)
			}
		})
	}
}

func TestNamesLowerInitial(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", want: ""},
		{name: "already lower", in: "greeter", want: "greeter"},
		{name: "single upper", in: "G", want: "g"},
		{name: "initial upper", in: "Greeter", want: "greeter"},
		{name: "leading acronym is preserved except first rune", in: "URLParser", want: "uRLParser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lowerInitial(tt.in)
			if got != tt.want {
				t.Fatalf("lowerInitial(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNamesLowerSnakeCase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", want: ""},
		{name: "lower word", in: "greeter", want: "greeter"},
		{name: "camel case", in: "SayHello", want: "say_hello"},
		{name: "acronym boundary", in: "HTTPServer", want: "http_server"},
		{name: "mixed separators", in: "foo.bar-Baz", want: "foo_bar_baz"},
		{name: "collapse separators", in: "Foo__Bar", want: "foo_bar"},
		{name: "trim separators", in: "_FooBar_", want: "foo_bar"},
		{name: "acronym with number", in: "HTTP2Server", want: "http2_server"},
		{name: "mixed acronym number", in: "SayHTTP2Server", want: "say_http2_server"},
		{name: "number in word", in: "IPv6Address", want: "ipv6_address"},
		{name: "digit boundary", in: "Foo2Bar", want: "foo2_bar"},
		{name: "leading digit", in: "2FAConfig", want: "2_fa_config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lowerSnakeCase(tt.in)
			if got != tt.want {
				t.Fatalf("lowerSnakeCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestPlanZeroValueHasNoIdentity(t *testing.T) {
	var file FilePlan
	if file.HasIdentity() {
		t.Fatalf("zero FilePlan must not have identity")
	}

	var service ServicePlan
	if service.HasIdentity() {
		t.Fatalf("zero ServicePlan must not have identity")
	}

	var method MethodPlan
	if method.HasIdentity() {
		t.Fatalf("zero MethodPlan must not have identity")
	}
}

func TestPlanStreamCapabilityUsesCapabilities(t *testing.T) {
	var zero StreamCapabilityContractPlan
	if !zero.IsZero() {
		t.Fatalf("zero capability contract must be zero: %+v", zero)
	}

	plan := StreamCapabilityContractPlan{
		CanSend:               true,
		FinishReturnsResponse: true,
	}

	if !plan.CanSend || plan.CanRecv || plan.CanCloseSend || !plan.FinishReturnsResponse {
		t.Fatalf("expected capability capabilities to be preserved: %+v", plan)
	}
	if plan.IsZero() {
		t.Fatalf("non-empty capability contract must not be zero: %+v", plan)
	}
}

func TestPlanMessageTransportConstants(t *testing.T) {
	tests := map[MessageTransport]string{
		MessageTransportConnect: "connect",
		MessageTransportGRPC:    "grpc",
	}

	for transport, want := range tests {
		if string(transport) != want {
			t.Fatalf("message transport = %q, want %q", transport, want)
		}
	}
}
