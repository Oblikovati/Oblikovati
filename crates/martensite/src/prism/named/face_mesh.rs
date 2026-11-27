// SPDX-License-Identifier: GPL-2.0-only
//! Named, complete derived face meshes for display, not geometric decisions (#4048).
use super::*;

/// Display triangles retained under the exact face reference that produced them.
/// Example: compare `key` with a selected face before coloring `triangles`.
#[derive(Clone, Debug, PartialEq)]
pub struct NamedFaceMesh {
    pub key: Vec<u8>,
    pub triangles: Vec<[[f64; 3]; 3]>,
    pub normals: Vec<[[f64; 3]; 3]>,
}

impl NamedSolid {
    /// Mesh a copied body with one complete, uniquely named group per face.
    /// Example: `solid.named_face_meshes(&history, document, options)?`.
    pub fn named_face_meshes(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        options: PrismMeshOptions,
    ) -> Result<Vec<NamedFaceMesh>, String> {
        let keys = self.face_reference_keys(history, document)?;
        let faces = keys
            .iter()
            .map(|key| self.native_face(history, document, key).cloned())
            .collect::<Result<Vec<_>, _>>()?;
        let display = super::super::mesh::named_face_display_triangles(
            self.native_shape(history, document)?,
            &faces,
            options,
            true,
        )?;
        Ok(keys
            .into_iter()
            .zip(display)
            .map(|(key, face)| NamedFaceMesh {
                key,
                triangles: face.iter().map(|tri| tri.positions).collect(),
                normals: face.into_iter().filter_map(|tri| tri.normals).collect(),
            })
            .collect())
    }
}
