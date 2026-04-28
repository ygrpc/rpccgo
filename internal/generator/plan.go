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
	Name         string
	GoName       string
	FullName     string
	Streaming    StreamingKind
	Request      MethodIOPlan
	Response     MethodIOPlan
	Lifecycle    LifecyclePlan
	NeedsCodec   bool
	RequestBody  []FieldPlan
	ResponseBody []FieldPlan
}

func (p MethodPlan) HasIdentity() bool {
	return p.Name != "" && p.GoName != "" && p.FullName != "" && p.Request.HasIdentity() && p.Response.HasIdentity()
}

type FieldPlan struct {
	Name       string
	GoName     string
	FullName   string
	Kind       string
	Repeated   bool
	Enum       bool
	Message    bool
	NativeType string
}

type LifecycleTerminalKind string

const (
	LifecycleTerminalFinishResult LifecycleTerminalKind = "finish_result"
	LifecycleTerminalOnDone       LifecycleTerminalKind = "on_done"
)

type LifecyclePlan struct {
	HasSend         bool
	HasFinish       bool
	HasCloseSend    bool
	HasCancel       bool
	CancelFinalizes bool
	TerminalKind    LifecycleTerminalKind
}

type MethodIOPlan struct {
	GoIdent  string
	FullName string
}

func (p MethodIOPlan) HasIdentity() bool {
	return p.GoIdent != "" && p.FullName != ""
}
