package interfaces

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestModuleRegistry(t *testing.T) {
	registry := NewModuleRegistry()

	// Test registration
	mockModule := &mockModule{name: "test"}
	err := registry.Register(mockModule)
	if err != nil {
		t.Errorf("Expected no error on registration, got %v", err)
	}

	// Test duplicate registration
	err = registry.Register(mockModule)
	if err == nil {
		t.Error("Expected error on duplicate registration")
	}

	// Test retrieval
	retrieved, exists := registry.Get("test")
	if !exists {
		t.Error("Expected module to exist")
	}
	if retrieved.Name() != "test" {
		t.Errorf("Expected name 'test', got %s", retrieved.Name())
	}

	// Test non-existent retrieval
	_, exists = registry.Get("nonexistent")
	if exists {
		t.Error("Expected non-existent module to not exist")
	}
}

// mockModule for testing
type mockModule struct {
	name string
}

func (m *mockModule) Name() string {
	return m.name
}

func (m *mockModule) Initialize(deps *ModuleDependencies) error {
	return nil
}

func (m *mockModule) RegisterRoutes(router *gin.RouterGroup) {}

func (m *mockModule) Shutdown() error {
	return nil
}
