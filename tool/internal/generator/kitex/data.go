package gen

// MethodInfo describes an RPC method.
type MethodInfo struct {
	Name         string // Method name
	RequestType  string // Request type name
	ResponseType string // Response type name
}

// GetName returns the method name.
func (m *MethodInfo) GetName() string {
	return m.Name
}

// GetRequestType returns the request type.
func (m *MethodInfo) GetRequestType() string {
	return m.RequestType
}

// GetResponseType returns the response type.
func (m *MethodInfo) GetResponseType() string {
	return m.ResponseType
}

// ScaffoldData is the root data structure passed to templates.
type ScaffoldData struct {
	Module       string
	Service      string
	ServiceLower string
	TypesModule  string
	UseTypes     string // Full import path from --use flag (e.g., github.com/org/proto/exmsg/user)
	GoVersion    string
	WithTrace    bool
	WithRedis    bool

	// Methods from proto file
	Methods []MethodInfo

	// Current loop iteration context (set during loop expansion)
	CurrentMethod *MethodInfo
}

// TypesImport returns the import path for types (messages).
// If UseTypes is set, uses it directly. Otherwise uses TypesModule/kitex_gen/ServiceLower.
func (d *ScaffoldData) TypesImport() string {
	if d.UseTypes != "" {
		return d.UseTypes
	}
	return d.TypesModule + "/kitex_gen/" + d.ServiceLower
}

// ServiceImport returns the import path for service client/server.
// If UseTypes is set, appends service name to it. Otherwise uses TypesModule/kitex_gen/ServiceLower/service.
func (d *ScaffoldData) ServiceImport() string {
	if d.UseTypes != "" {
		return d.UseTypes + "/" + d.ServiceLower + "service"
	}
	return d.TypesModule + "/kitex_gen/" + d.ServiceLower + "/" + d.ServiceLower + "service"
}

// MethodName returns the current method name for templates.
func (d *ScaffoldData) MethodName() string {
	if d.CurrentMethod != nil {
		return d.CurrentMethod.Name
	}
	return ""
}

// RequestType returns the current method's request type for templates.
func (d *ScaffoldData) RequestType() string {
	if d.CurrentMethod != nil {
		return d.CurrentMethod.RequestType
	}
	return ""
}

// ResponseType returns the current method's response type for templates.
func (d *ScaffoldData) ResponseType() string {
	if d.CurrentMethod != nil {
		return d.CurrentMethod.ResponseType
	}
	return ""
}

// WithMethod returns a copy with CurrentMethod set.
func (d *ScaffoldData) WithMethod(method *MethodInfo) *ScaffoldData {
	data := *d
	data.CurrentMethod = method
	return &data
}
