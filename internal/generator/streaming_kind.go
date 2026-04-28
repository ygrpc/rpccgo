package generator

type StreamingKind int

const (
	StreamingKindUnary StreamingKind = iota
	StreamingKindClientStreaming
	StreamingKindServerStreaming
	StreamingKindBidiStreaming
)

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
