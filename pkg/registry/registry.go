// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 Seqera
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
