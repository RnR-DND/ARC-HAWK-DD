package connectors

import (
	"fmt"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = map[string]ConnectorFactory{}
)

// Register adds a connector factory to the global registry.
// Called from sub-package init() functions.
func Register(sourceType string, factory ConnectorFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[sourceType] = factory
}

// Get returns a new Connector for the given source type.
func Get(sourceType string) (Connector, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[sourceType]
	if !ok {
		return nil, fmt.Errorf("no connector registered for source type %q", sourceType)
	}
	return factory(), nil
}

// Dispatch is an alias for Get, used by the orchestrator.
func Dispatch(sourceType string) (Connector, error) {
	return Get(sourceType)
}

// ListTypes returns all registered source types.
func ListTypes() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
