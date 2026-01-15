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
	GoVersion    string
	WithTrace    bool
	WithRedis    bool

	// Methods from proto file
	Methods []MethodInfo

	// Current loop iteration context (set during loop expansion)
	CurrentMethod *MethodInfo
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
