// SPDX-License-Identifier: GPL-2.0-only

package opfixture

import (
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// STLBytes tessellates a body at display quality and encodes it as binary STL — the import
// fixture five packages need to write an .stl file on disk and read it back.
//
// It lives here rather than in each test because meshio's encoders take a MESH, not a body
// (#2195: an exporter is a leaf that receives a mesh, it does not compute one). Without a
// shared fixture, every caller would repeat the tessellate-then-encode pair.
//
// Example: os.WriteFile(path, opfixture.STLBytes(body), 0o644)
func STLBytes(b *topo.Body) []byte {
	m, _ := tessellate.TessellateBody(b, tessellate.DefaultQuality())
	return meshio.EncodeBinarySTL(m)
}
