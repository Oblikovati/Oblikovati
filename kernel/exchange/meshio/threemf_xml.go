// SPDX-License-Identifier: GPL-2.0-only

package meshio

import (
	"bytes"
	"fmt"

	"oblikovati/kernel/ops"
)

// The OPC scaffolding 3MF requires: a content-types map and a root relationship pointing
// at the 3D model part. These are fixed for our single-object output (3MF core spec §2).
const (
	contentTypesXML = `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="model" ContentType="application/vnd.ms-package.3dmanufacturing-3dmodel+xml"/>
</Types>`

	relsXML = `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Target="/3D/3dmodel.model" Id="rel0" Type="http://schemas.microsoft.com/3dmanufacturing/2013/01/3dmodel"/>
</Relationships>`

	modelOpen = `<?xml version="1.0" encoding="UTF-8"?>
<model unit="millimeter" xml:lang="en-US" xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02">
<resources><object id="1" type="model"><mesh>`

	modelClose = `</mesh></object></resources>
<build><item objectid="1"/></build>
</model>`
)

// modelXML serializes a tessellated mesh into the 3MF <model> XML (one object).
func modelXML(mesh *ops.Mesh) []byte {
	var buf bytes.Buffer
	buf.WriteString(modelOpen)
	buf.WriteString("<vertices>")
	for _, p := range mesh.Positions {
		fmt.Fprintf(&buf, `<vertex x="%g" y="%g" z="%g"/>`, p.X, p.Y, p.Z)
	}
	buf.WriteString("</vertices><triangles>")
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		fmt.Fprintf(&buf, `<triangle v1="%d" v2="%d" v3="%d"/>`, mesh.Indices[t], mesh.Indices[t+1], mesh.Indices[t+2])
	}
	buf.WriteString("</triangles>")
	buf.WriteString(modelClose)
	return buf.Bytes()
}
