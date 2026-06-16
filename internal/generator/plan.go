package generator

// MessageTransport identifies the standard RPC transport selected for message contract generation.
type MessageTransport string

// Supported message transports for generated message artifacts.
const (
	MessageTransportConnect MessageTransport = "connect"
	MessageTransportGRPC    MessageTransport = "grpc"
)

// ServiceGenerationSelection records the rpccgo generation capabilities enabled for one service.
type ServiceGenerationSelection struct {
	MessageTransport MessageTransport
	NativeEnabled    bool
}

// HasIdentity reports whether the service generation selection has a valid transport identity.
func (s ServiceGenerationSelection) HasIdentity() bool {
	return s.MessageTransport == MessageTransportConnect || s.MessageTransport == MessageTransportGRPC
}

// GenerationPlan is the complete package-level plan produced from one protoc plugin request.
type GenerationPlan struct {
	Packages []PackagePlan
}

// PackagePlan groups files and shared generated artifacts by Go import path.
type PackagePlan struct {
	GoPackageName   string
	GoImportPath    string
	CGODir          string
	JNIClientDir    string
	JNIClass        string
	TopLevelSymbols []TopLevelSymbolPlan
	SharedArtifacts []GeneratedArtifactPlan
	Files           []FilePlan
}

// HasIdentity reports whether the package plan names a concrete Go package.
func (p PackagePlan) HasIdentity() bool {
	return p.GoPackageName != "" && p.GoImportPath != ""
}

// FilePlan describes the generated outputs and services for one protobuf file.
type FilePlan struct {
	GoPackageName           string
	GoImportPath            string
	ProtoPath               string
	GeneratedFilenamePrefix string
	CGODir                  string
	JNIClientDir            string
	JNIClass                string
	TopLevelSymbols         []TopLevelSymbolPlan
	Services                []ServicePlan
}

// HasIdentity reports whether the file plan names a protobuf file or contains services.
func (p FilePlan) HasIdentity() bool {
	return p.ProtoPath != "" || len(p.Services) > 0
}

// ServicePlan describes one protobuf service and the rpccgo artifacts generated for it.
type ServicePlan struct {
	Name       string
	GoName     string
	FullName   string
	DocComment string
	Generation ServiceGenerationSelection
	Methods    []MethodPlan
	Artifacts  []GeneratedArtifactPlan
}

// HasIdentity reports whether the service plan has protobuf identity and generation selection.
func (p ServicePlan) HasIdentity() bool {
	return p.Name != "" && p.GoName != "" && p.FullName != "" && p.Generation.HasIdentity()
}

// GeneratedArtifactKind identifies one family of generated rpccgo file.
type GeneratedArtifactKind string

// Generated artifact kinds supported by the renderer.
const (
	GeneratedArtifactKindRuntime          GeneratedArtifactKind = "runtime"
	GeneratedArtifactKindCodec            GeneratedArtifactKind = "codec"
	GeneratedArtifactKindNativeServer     GeneratedArtifactKind = "native_server"
	GeneratedArtifactKindCGONativeServer  GeneratedArtifactKind = "cgo_native_server"
	GeneratedArtifactKindCGONativeClient  GeneratedArtifactKind = "cgo_native_client"
	GeneratedArtifactKindMessageServer    GeneratedArtifactKind = "message_server"
	GeneratedArtifactKindCGOMessageServer GeneratedArtifactKind = "cgo_message_server"
	GeneratedArtifactKindCGOMessageClient GeneratedArtifactKind = "cgo_message_client"
	GeneratedArtifactKindJNIMessageClient GeneratedArtifactKind = "jni_message_client"
	GeneratedArtifactKindJNIKotlinClient  GeneratedArtifactKind = "jni_kotlin_client"
	GeneratedArtifactKindSharedCGOExports GeneratedArtifactKind = "shared_cgo_exports"
	GeneratedArtifactKindSharedCGOMain    GeneratedArtifactKind = "shared_cgo_main"
)

// GeneratedArtifactPlan records the kind and output filename of one generated artifact.
type GeneratedArtifactPlan struct {
	Kind     GeneratedArtifactKind
	Filename string
}

// Artifact returns the generated artifact plan of the requested kind for the service.
func (p ServicePlan) Artifact(kind GeneratedArtifactKind) (GeneratedArtifactPlan, bool) {
	for _, artifact := range p.Artifacts {
		if artifact.Kind == kind {
			return artifact, true
		}
	}
	return GeneratedArtifactPlan{}, false
}

// HasArtifact reports whether the service generates an artifact of the requested kind.
func (p ServicePlan) HasArtifact(kind GeneratedArtifactKind) bool {
	_, ok := p.Artifact(kind)
	return ok
}

// TopLevelSymbolKind identifies protobuf symbols that can collide with generated Go names.
type TopLevelSymbolKind string

// Top-level symbol kinds tracked for generated symbol collision checks.
const (
	TopLevelSymbolKindMessage TopLevelSymbolKind = "message"
	TopLevelSymbolKindEnum    TopLevelSymbolKind = "enum"
	TopLevelSymbolKindService TopLevelSymbolKind = "service"
)

// TopLevelSymbolPlan records a package-level protobuf symbol visible to generated code.
type TopLevelSymbolPlan struct {
	GoName   string
	FullName string
	Kind     TopLevelSymbolKind
}

// MethodPlan describes one protobuf method and its contract and render projections.
type MethodPlan struct {
	Name       string
	GoName     string
	FullName   string
	DocComment string
	Streaming  StreamingKind
	Request    MethodIOPlan
	Response   MethodIOPlan
	Contract   MethodContractPlan
	RenderPlan MethodRenderPlan
}

// HasIdentity reports whether the method plan has protobuf identity and request/response types.
func (p MethodPlan) HasIdentity() bool {
	return p.Name != "" && p.GoName != "" && p.FullName != "" && p.Request.HasIdentity() && p.Response.HasIdentity()
}

// FieldPlan describes one top-level protobuf request or response field used by native projection.
type FieldPlan struct {
	Name     string
	GoName   string
	FullName string
	Kind     FieldKind
	Repeated bool
	Enum     bool
	Message  bool
	EnumType MethodIOPlan
	Native   NativeFieldPlan
}

// FieldKind classifies protobuf field kinds used by contract planning.
type FieldKind string

// Field kinds supported by rpccgo contract planning.
const (
	FieldKindSignedInt32   FieldKind = "signed_int32"
	FieldKindSignedInt64   FieldKind = "signed_int64"
	FieldKindUnsignedInt32 FieldKind = "unsigned_int32"
	FieldKindUnsignedInt64 FieldKind = "unsigned_int64"
	FieldKindFloat         FieldKind = "float"
	FieldKindDouble        FieldKind = "double"
	FieldKindBool          FieldKind = "bool"
	FieldKindString        FieldKind = "string"
	FieldKindBytes         FieldKind = "bytes"
	FieldKindMessage       FieldKind = "message"
	FieldKindEnum          FieldKind = "enum"
)

// NativeFieldKind classifies the Go native boundary representation for a field.
type NativeFieldKind string

// Native field kinds supported by native contract planning.
const (
	NativeFieldKindSignedNumeric NativeFieldKind = "signed_numeric"
	NativeFieldKindFloat         NativeFieldKind = "float"
	NativeFieldKindBool          NativeFieldKind = "bool"
	NativeFieldKindString        NativeFieldKind = "string"
	NativeFieldKindBytes         NativeFieldKind = "bytes"
	NativeFieldKindMessageBytes  NativeFieldKind = "message_bytes"
	NativeFieldKindEnum          NativeFieldKind = "enum"
)

// NativeABIShape identifies the C ABI shape derived from a native field.
type NativeABIShape string

// Native ABI shapes supported by generated C projection.
const (
	NativeABIShapeScalar                NativeABIShape = "scalar"
	NativeABIShapeRepeated              NativeABIShape = "repeated"
	NativeABIShapeBoolByte              NativeABIShape = "bool_byte"
	NativeABIShapeBoolByteBufferWrapper NativeABIShape = "bool_byte_buffer_wrapper"
	NativeABIShapeMessageBytes          NativeABIShape = "message_bytes"
)

// NativeFieldPlan records the native kind and ABI shape for one protobuf field.
type NativeFieldPlan struct {
	Kind  NativeFieldKind
	Shape NativeABIShape
}

// MethodContractPlan groups the native, message, and stream contract plans for one method.
type MethodContractPlan struct {
	Native  NativeContractPlan
	Message MessageContractPlan
	Stream  StreamCapabilityContractPlan
}

// NativeContractPlan records flat native request and response fields for one method.
type NativeContractPlan struct {
	RequestFields  []FieldPlan
	ResponseFields []FieldPlan
}

// MessageContractPlan records typed protobuf request and response message boundaries.
type MessageContractPlan struct {
	RequestType  MethodIOPlan
	ResponseType MethodIOPlan
}

// StreamCapabilityContractPlan records the operations generated for a streaming method.
type StreamCapabilityContractPlan struct {
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
}

// IsZero reports whether the stream capability has no streaming operations enabled.
func (p StreamCapabilityContractPlan) IsZero() bool {
	return !p.CanSend && !p.CanRecv && !p.CanCloseSend && !p.FinishReturnsResponse
}

// MethodIOPlan describes a protobuf request or response message type used by a method.
type MethodIOPlan struct {
	GoName       string
	GoImportPath string
	FullName     string
}

// HasIdentity reports whether the method IO plan names a concrete protobuf message.
func (p MethodIOPlan) HasIdentity() bool {
	return p.GoName != "" && p.FullName != ""
}
