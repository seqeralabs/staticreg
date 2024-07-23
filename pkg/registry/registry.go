package registry

import (
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/seqeralabs/staticreg/pkg/cfg"
)

func hostFromConfig(rootCfg cfg.Root) config.Host {
	regHost := config.Host{
		Name:     rootCfg.RegistryHostname,
		Hostname: rootCfg.RegistryHostname,
		User:     rootCfg.RegistryUser,
		Pass:     rootCfg.RegistryPassword,
	}

	if !rootCfg.TLSEnabled {
		regHost.TLS = config.TLSDisabled
	}

	if rootCfg.SkipTLSVerify {
		regHost.TLS = config.TLSInsecure
	}

	return regHost
}

func ClientFromConfig(rootCfg cfg.Root) *regclient.RegClient {
	regHost := hostFromConfig(rootCfg)
	return regclient.New(
		regclient.WithConfigHost(regHost),
		regclient.WithUserAgent("seqera/staticreg"),
	)
}
