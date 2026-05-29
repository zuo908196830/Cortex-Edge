// Package regiester 保留旧的拼写错误导入路径，避免已有代码直接失效。
package regiester

import "edge-gateway/registry"

type Registry = registry.Registry
type MemoryRegistry = registry.MemoryRegistry

func NewMemoryRegistry() *MemoryRegistry {
	return registry.NewMemoryRegistry()
}
