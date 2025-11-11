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
package webhook

import (
	"strings"
	"time"
)

// DistributionEventEnvelope represents the envelope containing Docker Distribution events
type DistributionEventEnvelope struct {
	Events []DistributionEvent `json:"events"`
}

// DistributionEvent represents a single Docker Distribution event
type DistributionEvent struct {
	ID        string                   `json:"id"`
	Timestamp time.Time                `json:"timestamp"`
	Action    string                   `json:"action"`
	Target    DistributionEventTarget  `json:"target"`
	Request   DistributionEventRequest `json:"request"`
	Actor     DistributionEventActor   `json:"actor"`
	Source    DistributionEventSource  `json:"source"`
}

// DistributionEventTarget contains details about the affected artifact
type DistributionEventTarget struct {
	MediaType  string `json:"mediaType"`
	Size       int64  `json:"size,omitempty"`
	Digest     string `json:"digest"`
	Length     int64  `json:"length,omitempty"`
	Repository string `json:"repository"`
	URL        string `json:"url,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// DistributionEventRequest contains request metadata
type DistributionEventRequest struct {
	ID        string `json:"id"`
	Addr      string `json:"addr"`
	Host      string `json:"host"`
	Method    string `json:"method"`
	UserAgent string `json:"useragent"`
}

// DistributionEventActor represents the agent that initiated the event
type DistributionEventActor struct {
	Name string `json:"name,omitempty"`
}

// DistributionEventSource contains information about the registry node
type DistributionEventSource struct {
	Addr       string `json:"addr"`
	InstanceID string `json:"instanceID"`
}

// IsPushEvent checks if the event is a push action
func (e *DistributionEvent) IsPushEvent() bool {
	return e.Action == "push"
}

// IsPullEvent checks if the event is a pull action
func (e *DistributionEvent) IsPullEvent() bool {
	return e.Action == "pull"
}

// IsManifestPush checks if the event is a manifest push (final step of container push)
func (e *DistributionEvent) IsManifestPush() bool {
	return e.IsPushEvent() && (e.Target.MediaType == "application/vnd.docker.distribution.manifest.v2+json" ||
		e.Target.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		e.Target.MediaType == "application/vnd.oci.image.manifest.v1+json" ||
		e.Target.MediaType == "application/vnd.oci.image.index.v1+json")
}

// IsManifestPull checks if the event is a manifest pull (final step of container pull)
func (e *DistributionEvent) IsManifestPull() bool {
	return e.IsPullEvent() && e.Target.Tag != "" && (e.Target.MediaType == "application/vnd.docker.distribution.manifest.v2+json" ||
		e.Target.MediaType == "application/vnd.docker.distribution.manifest.list.v2+json" ||
		e.Target.MediaType == "application/vnd.oci.image.manifest.v1+json" ||
		e.Target.MediaType == "application/vnd.oci.image.index.v1+json")
}

// IsFromStaticReg checks if the event originates from staticreg itself
// This helps exclude staticreg's internal manifest fetches from pull metrics
func (e *DistributionEvent) IsFromStaticReg() bool {
	return strings.Contains(e.Request.UserAgent, "seqera/staticreg")
}

// GetArchitecture extracts architecture from the event's user agent string
// Docker clients typically include platform info in the user agent
// Format examples:
//   - "docker/20.10.7 go/go1.16.4 git-commit/f0df350 kernel/5.10.0 os/linux arch/amd64"
//   - "containerd/1.4.4+unknown"
func (e *DistributionEvent) GetArchitecture() string {
	userAgent := e.Request.UserAgent
	if userAgent == "" {
		return "unknown"
	}

	// Try to extract architecture from user agent
	// Look for "arch/xxx" pattern
	parts := strings.Split(userAgent, " ")
	for _, part := range parts {
		if strings.HasPrefix(part, "arch/") {
			arch := strings.TrimPrefix(part, "arch/")
			return arch
		}
	}

	// If no arch found, try to infer from other patterns
	// containerd and other clients may not include arch
	return "unknown"
}
