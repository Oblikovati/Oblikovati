// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/exchange"
	"oblikovati.org/model/pointcloud"
)

// Point-cloud import (M17-F06, #645): the shared attach path used by both the Insert ▸ Import
// Point Cloud command (via the head's file dialog) and the pointClouds.attach wire method. The
// read → embed → decode → attach sequence lives in the model/exchange dispatch vertical
// (exchange.ImportPointCloud, #1646) so scan imports share the standard exchange seam.

// AttachPointCloud reads the scan file at fullFileName, embeds its bytes in the active part's
// resource table (ADR-0031), decodes its points into the document working unit (#1636), and
// attaches the cloud under name (a unique name is minted when name is empty). Warnings report
// skipped malformed records (#1646). Errors when there is no active part, the file is
// unreadable, or the scan format has no reader.
func (s *Session) AttachPointCloud(name, fullFileName string) (*pointcloud.PointCloud, []string, error) {
	part, err := activePart(s)
	if err != nil {
		return nil, nil, err
	}
	if fullFileName == "" {
		return nil, nil, fmt.Errorf("app: AttachPointCloud needs a file name")
	}
	pc, res, err := exchange.ImportPointCloud(part, name, fullFileName)
	if err != nil {
		return nil, nil, err
	}
	return pc, res.Warnings, nil
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
