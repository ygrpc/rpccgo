package generator

type AdapterToken string

const (
	AdapterTokenMessageConnect AdapterToken = "msg-connect"
	AdapterTokenMessageGRPC    AdapterToken = "msg-grpc"
	AdapterTokenNative         AdapterToken = "native"
)

type AdapterSelection struct {
	Tokens []AdapterToken
}

func (s AdapterSelection) HasTokens() bool {
	return len(s.Tokens) > 0
}

func (s AdapterSelection) Has(token AdapterToken) bool {
	for _, current := range s.Tokens {
		if current == token {
			return true
		}
	}
	return false
}

type FilePlan struct {
	GoPackageName string
	GoImportPath  string
	ProtoPath     string
	Services      []ServicePlan
}

func (p FilePlan) HasIdentity() bool {
	return p.ProtoPath != "" || len(p.Services) > 0
}

type ServicePlan struct {
	Name       string
	GoName     string
	FullName   string
	Adapters   AdapterSelection
	Methods    []MethodPlan
	NeedsCodec bool
}

func (p ServicePlan) HasIdentity() bool {
	return p.Name != "" && p.GoName != "" && p.FullName != "" && p.Adapters.HasTokens()
}

type MethodPlan struct {
	Name            string
	GoName          string
	FullName        string
	Streaming       StreamingKind
	Request         MethodIOPlan
	Response        MethodIOPlan
	NativeContract  NativeContractPlan
	MessageContract MessageContractPlan
	Lifecycle       LifecyclePlan
	NeedsCodec      bool
	RequestBody     []FieldPlan
	ResponseBody    []FieldPlan
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
	Native   NativeFieldPlan
}

type FieldKind string

const (
	FieldKindSignedInt32 FieldKind = "signed_int32"
	FieldKindSignedInt64 FieldKind = "signed_int64"
	FieldKindFloat       FieldKind = "float"
	FieldKindDouble      FieldKind = "double"
	FieldKindBool        FieldKind = "bool"
	FieldKindString      FieldKind = "string"
	FieldKindBytes       FieldKind = "bytes"
	FieldKindMessage     FieldKind = "message"
	FieldKindEnum        FieldKind = "enum"
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

type NativeContractPlan struct {
	RequestFields  []FieldPlan
	ResponseFields []FieldPlan
}

type MessageContractPlan struct {
	RequestType  MethodIOPlan
	ResponseType MethodIOPlan
}

type LifecycleTerminalKind string

const (
	LifecycleTerminalFinishResult LifecycleTerminalKind = "finish_result"
	LifecycleTerminalOnDone       LifecycleTerminalKind = "on_done"
)

type LifecyclePlan struct {
	HasStart        bool
	HasSend         bool
	HasFinish       bool
	HasCloseSend    bool
	HasCancel       bool
	CancelFinalizes bool
	HasOnRead       bool
	HasOnDone       bool
	TerminalKind    LifecycleTerminalKind
}

type MethodIOPlan struct {
	GoName       string
	GoImportPath string
	FullName     string
}

func (p MethodIOPlan) HasIdentity() bool {
	return p.GoName != "" && p.FullName != ""
}
