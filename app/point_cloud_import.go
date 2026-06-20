// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"os"

	"oblikovati.org/model/doc"
	"oblikovati.org/model/pointcloud"
)

// Point-cloud import (M17-F06, #645): the shared attach path used by both the Insert ▸ Import
// Point Cloud command (via the head's file dialog) and the pointClouds.attach wire method, so the
// read → embed → decode → attach sequence lives in one place.

// AttachPointCloud reads the scan file at fullFileName, embeds its bytes in the active part's
// resource table (ADR-0031), decodes its points, and attaches the cloud under name (a unique name
// is minted when name is empty). Errors when there is no active part, the file is unreadable, or
// the scan format has no reader.
func (s *Session) AttachPointCloud(name, fullFileName string) (*pointcloud.PointCloud, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, err
	}
	if fullFileName == "" {
		return nil, fmt.Errorf("app: AttachPointCloud needs a file name")
	}
	data, err := os.ReadFile(fullFileName)
	if err != nil {
		return nil, fmt.Errorf("app: read scan %q: %w", fullFileName, err)
	}
	points, err := pointcloud.ReadScan(fullFileName, data)
	if err != nil {
		return nil, err
	}
	if name == "" {
		name = part.PointClouds().UniqueName("Cloud")
	}
	rid := part.AddResource(doc.Resource{Type: "PointCloudScan", Encoding: doc.EncodingUTF8, Value: data, Origin: fullFileName})
	return part.PointClouds().Add(name, fullFileName, rid, points)
}

// RequestImportPointCloud flags that the user asked to import a scan; the head opens its file
// dialog and TakeImportPointCloudRequest consumes the flag (one-shot, so the dialog opens once).
func (s *Session) RequestImportPointCloud() { s.pointCloudRequested = true }

// TakeImportPointCloudRequest returns and clears the pending import-point-cloud request.
func (s *Session) TakeImportPointCloudRequest() bool {
	req := s.pointCloudRequested
	s.pointCloudRequested = false
	return req
}
