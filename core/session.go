package core

// Session 保存与单个远端节点通信时的运行上下文，不包含长期业务状态。
type Session struct {
	NodeID       string
	Endpoint     string
	Adapter      string      // 适配器类型（php / mock / …）
	Transport    string      // 传输类型（httpform / mock / …）
	Codec        string      // 编解码类型
	Envelope     string      // 边界协议类型
	Capabilities []Capability
	Metadata     map[string]string
}

// NewSession 构建一个 map 已初始化的 Session。
func NewSession(nodeID, endpoint string) *Session {
	return &Session{
		NodeID:       nodeID,
		Endpoint:     endpoint,
		Capabilities: make([]Capability, 0),
		Metadata:     make(map[string]string),
	}
}

// HasCapability 判断 Session 是否支持给定能力。
func (s *Session) HasCapability(c Capability) bool {
	for _, cap := range s.Capabilities {
		if cap == c {
			return true
		}
	}
	return false
}
