package generator

type MessageTransport string

const (
	MessageTransportConnect MessageTransport = "connect"
	MessageTransportGRPC    MessageTransport = "grpc"
)

type ServiceGenerationSelection struct {
	MessageTransport MessageTransport
	NativeEnabled    bool
}

func (s ServiceGenerationSelection) HasIdentity() bool {
	return s.MessageTransport == MessageTransportConnect || s.MessageTransport == MessageTransportGRPC
}

type GenerationPlan struct {
	Packages []PackagePlan
}

type PackagePlan struct {
	GoPackageName   string
	GoImportPath    string
	CGODir          string
	TopLevelSymbols []TopLevelSymbolPlan
	SharedArtifacts []GeneratedArtifactPlan
	Files           []FilePlan
}

func (p PackagePlan) HasIdentity() bool {
	return p.GoPackageName != "" && p.GoImportPath != ""
}

type FilePlan struct {
	GoPackageName           string
	GoImportPath            string
	ProtoPath               string
	GeneratedFilenamePrefix string
	CGODir                  string
	TopLevelSymbols         []TopLevelSymbolPlan
	Services                []ServicePlan
}

func (p FilePlan) HasIdentity() bool {
	return p.ProtoPath != "" || len(p.Services) > 0
}

type ServicePlan struct {
	Name       string
	GoName     string
	FullName   string
	DocComment string
	Generation ServiceGenerationSelection
	Methods    []MethodPlan
	Artifacts  []GeneratedArtifactPlan
}

func (p ServicePlan) HasIdentity() bool {
	return p.Name != "" && p.GoName != "" && p.FullName != "" && p.Generation.HasIdentity()
}

type GeneratedArtifactKind string

const (
	GeneratedArtifactKindRuntime          GeneratedArtifactKind = "runtime"
	GeneratedArtifactKindCodec            GeneratedArtifactKind = "codec"
	GeneratedArtifactKindNativeServer     GeneratedArtifactKind = "native_server"
	GeneratedArtifactKindCGONativeServer  GeneratedArtifactKind = "cgo_native_server"
	GeneratedArtifactKindCGONativeClient  GeneratedArtifactKind = "cgo_native_client"
	GeneratedArtifactKindMessageServer    GeneratedArtifactKind = "message_server"
	GeneratedArtifactKindCGOMessageServer GeneratedArtifactKind = "cgo_message_server"
	GeneratedArtifactKindCGOMessageClient GeneratedArtifactKind = "cgo_message_client"
	GeneratedArtifactKindSharedCGOExports GeneratedArtifactKind = "shared_cgo_exports"
)

type GeneratedArtifactPlan struct {
	Kind     GeneratedArtifactKind
	Filename string
}

func (p ServicePlan) Artifact(kind GeneratedArtifactKind) (GeneratedArtifactPlan, bool) {
	for _, artifact := range p.Artifacts {
		if artifact.Kind == kind {
			return artifact, true
		}
	}
	return GeneratedArtifactPlan{}, false
}

func (p ServicePlan) HasArtifact(kind GeneratedArtifactKind) bool {
	_, ok := p.Artifact(kind)
	return ok
}

type TopLevelSymbolKind string

const (
	TopLevelSymbolKindMessage TopLevelSymbolKind = "message"
	TopLevelSymbolKindEnum    TopLevelSymbolKind = "enum"
	TopLevelSymbolKindService TopLevelSymbolKind = "service"
)

type TopLevelSymbolPlan struct {
	GoName   string
	FullName string
	Kind     TopLevelSymbolKind
}

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

func (p MethodPlan) HasIdentity() bool {
	return p.Name != "" && p.GoName != "" && p.FullName != "" && p.Request.HasIdentity() && p.Response.HasIdentity()
}

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

type FieldKind string

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

type NativeFieldKind string

const (
	NativeFieldKindSignedNumeric NativeFieldKind = "signed_numeric"
	NativeFieldKindFloat         NativeFieldKind = "float"
	NativeFieldKindBool          NativeFieldKind = "bool"
	NativeFieldKindString        NativeFieldKind = "string"
	NativeFieldKindBytes         NativeFieldKind = "bytes"
	NativeFieldKindMessageBytes  NativeFieldKind = "message_bytes"
	NativeFieldKindEnum          NativeFieldKind = "enum"
)

type NativeABIShape string

const (
	NativeABIShapeScalar                NativeABIShape = "scalar"
	NativeABIShapeRepeated              NativeABIShape = "repeated"
	NativeABIShapeBoolByte              NativeABIShape = "bool_byte"
	NativeABIShapeBoolByteBufferWrapper NativeABIShape = "bool_byte_buffer_wrapper"
	NativeABIShapeMessageBytes          NativeABIShape = "message_bytes"
)

type NativeFieldPlan struct {
	Kind  NativeFieldKind
	Shape NativeABIShape
}

type MethodContractPlan struct {
	Native  NativeContractPlan
	Message MessageContractPlan
	Stream  StreamCapabilityContractPlan
}

type NativeContractPlan struct {
	RequestFields  []FieldPlan
	ResponseFields []FieldPlan
}

type MessageContractPlan struct {
	RequestType  MethodIOPlan
	ResponseType MethodIOPlan
}

type StreamCapabilityContractPlan struct {
	CanSend               bool
	CanRecv               bool
	CanCloseSend          bool
	FinishReturnsResponse bool
}

func (p StreamCapabilityContractPlan) IsZero() bool {
	return !p.CanSend && !p.CanRecv && !p.CanCloseSend && !p.FinishReturnsResponse
}

type MethodIOPlan struct {
	GoName       string
	GoImportPath string
	FullName     string
}

func (p MethodIOPlan) HasIdentity() bool {
	return p.GoName != "" && p.FullName != ""
}
