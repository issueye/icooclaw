// Package channels provides channel management for icooclaw.
package channels

import (
	"sync"
)

// Factory creates Channel instances.
type Factory func(config map[string]any) (Channel, error)

var (
	factoriesMu sync.RWMutex
	factories   = make(map[string]Factory)
)

// RegisterFactory registers a channel factory.
func RegisterFactory(name string, f Factory) {
	factoriesMu.Lock()
	defer factoriesMu.Unlock()
	factories[name] = f
}

// GetFactory gets a channel factory by name.
func GetFactory(name string) (Factory, bool) {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()
	f, ok := factories[name]
	return f, ok
}

// ListFactories lists all registered factory names.
func ListFactories() []string {
	factoriesMu.RLock()
	defer factoriesMu.RUnlock()

	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	return names
}