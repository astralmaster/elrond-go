package factory

import (
	"github.com/astralmaster/elrond-go/config"
	"github.com/astralmaster/elrond-go/debug/resolver"
)

// NewInterceptorResolverDebuggerFactory will instantiate an InterceptorResolverDebugHandler based on the provided config
func NewInterceptorResolverDebuggerFactory(config config.InterceptorResolverDebugConfig) (InterceptorResolverDebugHandler, error) {
	if !config.Enabled {
		return resolver.NewDisabledInterceptorResolver(), nil
	}

	return resolver.NewInterceptorResolver(config)
}
