package interfaces

import (
	"database/sql"

	"github.com/arc-platform/backend/modules/shared/config"
	"github.com/arc-platform/backend/modules/shared/infrastructure/persistence"
	"github.com/arc-platform/backend/modules/shared/infrastructure/vault"
	"github.com/gin-gonic/gin"
)

// Module represents a modular component of the application
type Module interface {
	// Name returns the module name
	Name() string

	// Initialize sets up the module with its dependencies
	Initialize(deps *ModuleDependencies) error

	// RegisterRoutes registers the module's HTTP routes
	RegisterRoutes(router *gin.RouterGroup)

	// Shutdown performs cleanup when the application stops
	Shutdown() error
}

// ModuleDependencies contains shared dependencies for all modules
type ModuleDependencies struct {
	// Database connection
	DB *sql.DB

	// Neo4j repository for graph operations
	Neo4jRepo *persistence.Neo4jRepository

	// Application configuration
	Config *config.Config

	// Module registry for inter-module communication
	Registry *ModuleRegistry

	// WebSocket service for real-time communication
	WebSocketService interface{}

	// Interface dependencies (injected by main.go for loose coupling)
	AssetManager     AssetManager
	FindingsProvider FindingsProvider
	LineageSync      LineageSync
	AuditLogger      AuditLogger
	MemoryRecorder   MemoryRecorder

	// Vault client for credential lifecycle management (nil when disabled)
	VaultClient *vault.Client
}

// ModuleRegistry manages all registered modules
type ModuleRegistry struct {
	modules map[string]Module
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
	}
}

// Register adds a module to the registry
func (r *ModuleRegistry) Register(module Module) error {
	name := module.Name()
	if _, exists := r.modules[name]; exists {
		return ErrModuleAlreadyRegistered{Name: name}
	}
	r.modules[name] = module
	return nil
}

// Get retrieves a module by name
func (r *ModuleRegistry) Get(name string) (Module, bool) {
	module, exists := r.modules[name]
	return module, exists
}

// GetAll returns all registered modules
func (r *ModuleRegistry) GetAll() []Module {
	modules := make([]Module, 0, len(r.modules))
	for _, module := range r.modules {
		modules = append(modules, module)
	}
	return modules
}

// InitializeAll initializes all registered modules
func (r *ModuleRegistry) InitializeAll(deps *ModuleDependencies) error {
	for _, module := range r.modules {
		if err := module.Initialize(deps); err != nil {
			return ErrModuleInitialization{
				ModuleName: module.Name(),
				Err:        err,
			}
		}
	}
	return nil
}

// ShutdownAll shuts down all registered modules
func (r *ModuleRegistry) ShutdownAll() error {
	for _, module := range r.modules {
		if err := module.Shutdown(); err != nil {
			return ErrModuleShutdown{
				ModuleName: module.Name(),
				Err:        err,
			}
		}
	}
	return nil
}

// Errors

type ErrModuleAlreadyRegistered struct {
	Name string
}

func (e ErrModuleAlreadyRegistered) Error() string {
	return "module already registered: " + e.Name
}

type ErrModuleInitialization struct {
	ModuleName string
	Err        error
}

func (e ErrModuleInitialization) Error() string {
	return "failed to initialize module " + e.ModuleName + ": " + e.Err.Error()
}

type ErrModuleShutdown struct {
	ModuleName string
	Err        error
}

func (e ErrModuleShutdown) Error() string {
	return "failed to shutdown module " + e.ModuleName + ": " + e.Err.Error()
}
