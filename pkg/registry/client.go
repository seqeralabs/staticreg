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
	"context"
	"html/template"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

type RepoData struct {
	Name          string
	PullReference string
	LastUpdatedAt time.Time
}

type ImageInfo struct {
	Image         v1.Image
	Reference     string
	Architectures []string
	ScanUrl       template.HTML
	InspectUrl    template.HTML
}

// Client interface defines methods for interacting with a container registry
type Client interface {
	// RepoList retrieves a list of repository names from the registry
	RepoList(ctx context.Context) (repos map[string]RepoData, err error)

	// TagList retrieves a list of tags for a specified repository
	TagList(ctx context.Context, repo string) (tags []string, err error)

	// ImageInfo retrieves detailed information about a specific image identified by its repository and tag
	ImageInfo(ctx context.Context, repo string, tag string) (imageInfo ImageInfo, err error)
}

// GetImageCreationTime extracts the creation time from an image, checking both
// the config file and manifest annotations for OCI artifacts
func GetImageCreationTime(image v1.Image) time.Time {
	cfg, err := image.ConfigFile()
	if err == nil && !cfg.Created.IsZero() {
		return cfg.Created.Time
	}

	manifest, err := image.Manifest()
	if err == nil && manifest.Annotations != nil {
		if createdAnnotation, exists := manifest.Annotations["org.opencontainers.image.created"]; exists {
			if parsedTime, err := time.Parse(time.RFC3339, createdAnnotation); err == nil {
				return parsedTime
			}
		}
	}

	return time.Time{}
}
