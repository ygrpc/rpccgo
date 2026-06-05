package generator

// StreamingKind identifies the unary or streaming shape of a protobuf method.
type StreamingKind int

// Streaming kinds derived from protobuf client/server streaming flags.
const (
	StreamingKindUnary StreamingKind = iota
	StreamingKindClientStreaming
	StreamingKindServerStreaming
	StreamingKindBidiStreaming
)

// StreamingKindOf converts protobuf client/server streaming flags into a StreamingKind.
func StreamingKindOf(isClientStream, isServerStream bool) StreamingKind {
	switch {
	case isClientStream && isServerStream:
		return StreamingKindBidiStreaming
	case isClientStream:
		return StreamingKindClientStreaming
	case isServerStream:
		return StreamingKindServerStreaming
	default:
		return StreamingKindUnary
	}
}
