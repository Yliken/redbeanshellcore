package core

// Adapter defines the contract every adapter must satisfy.
type Adapter interface {
	Capabilities() []Capability
	HasCapability(cap Capability) bool
	WrapOp(operation Operation) (Operation, error)
}
