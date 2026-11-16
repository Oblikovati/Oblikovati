// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbrep::{brep::BRepTool, topods::Shape};
use martensite_tkg3d::top_abs::Orientation;
use martensite_tkmath::poly::{Triangulation, mesh_purpose};
use martensite_tkmesh::brep_mesh::{BRepMeshDiscretRoot, BRepMeshIncrementalMesh};
/// Absolute display approximation limits in working units and radians; these are not modeling tolerances.
#[derive(Clone, Copy, Debug)]
pub struct PrismMeshOptions {
    pub chord_error: f64,
    pub angular_error: f64,
}
#[derive(Clone, Debug, PartialEq)]
pub struct PrismFaceMesh {
    pub role: PrismFaceRole,
    pub triangles: Vec<[[f64; 3]; 3]>,
}
impl PrismFaceMesh {
    /// The face's outline, for display: facet sides only one of this face's facets uses.
    /// Example: `face.boundary_segments()` gives a square's four sides for its two triangles.
    pub fn boundary_segments(&self) -> Vec<[[f64; 3]; 2]> {
        boundary_segments(&self.triangles)
    }
}
fn boundary_segments(triangles: &[[[f64; 3]; 3]]) -> Vec<[[f64; 3]; 2]> {
    let key = |point: [f64; 3]| point.map(f64::to_bits);
    let mut uses = std::collections::BTreeMap::new();
    for [a, b, c] in triangles {
        for (from, to) in [(a, b), (b, c), (c, a)] {
            let (low, high) = (key(*from).min(key(*to)), key(*from).max(key(*to)));
            uses.entry((low, high)).or_insert(([*from, *to], 0)).1 += 1;
        }
    }
    uses.into_values()
        .filter(|(_, count)| *count == 1)
        .map(|(segment, _)| segment)
        .collect()
}
/// Derived display geometry. Construction roles are not persistent pick references.
#[derive(Clone, Debug, PartialEq)]
pub struct PrismPresentation {
    pub faces: Vec<PrismFaceMesh>,
}
pub(super) fn derive_presentation(
    snapshot: &PrismProbeSnapshot,
    options: PrismMeshOptions,
) -> Result<PrismPresentation, String> {
    validate_options(options)?;
    let (shape, faces) = snapshot::copied_prism(&snapshot.shape, &snapshot.faces)
        .map_err(|cause| cause.to_string())?;
    mesh_shape(&shape, options)?;
    let body_faces = snapshot::BodyFaceIndex::new(&shape)?;
    let faces = faces
        .iter()
        .map(|(role, lineage)| {
            let body_face = body_faces
                .resolve(lineage)
                .map_err(|cause| format!("prism {role:?} face: {cause}"))?;
            face_mesh(*role, &body_face)
        })
        .collect::<Result<Vec<_>, _>>()?;
    Ok(PrismPresentation { faces })
}
fn mesh_shape(shape: &Shape, options: PrismMeshOptions) -> Result<(), String> {
    let mesher = BRepMeshIncrementalMesh::with_shape(
        shape,
        options.chord_error,
        false,
        options.angular_error,
        false,
    )
    .map_err(|error| {
        format!("prism mesh {options:?}: expected completed kernel meshing: {error}")
    })?;
    if !mesher.is_done() || mesher.get_status_flags() != 0 {
        return Err(format!(
            "prism mesh done={}, flags={}: expected completed meshing without defects",
            mesher.is_done(),
            mesher.get_status_flags()
        ));
    }
    Ok(())
}
fn validate_options(options: PrismMeshOptions) -> Result<(), String> {
    if !options.chord_error.is_finite()
        || options.chord_error <= 0.
        || !options.angular_error.is_finite()
        || options.angular_error <= 0.
    {
        return Err(format!(
            "prism mesh {options:?}: expected finite positive chord and angular errors"
        ));
    }
    Ok(())
}
fn face_mesh(role: PrismFaceRole, shape: &Shape) -> Result<PrismFaceMesh, String> {
    let triangles = face_triangles(shape, &format!("prism {role:?}"))?;
    Ok(PrismFaceMesh { role, triangles })
}
fn face_triangles(shape: &Shape, context: &str) -> Result<Vec<[[f64; 3]; 3]>, String> {
    let face = martensite_tkbrep::topods::face(shape)
        .map_err(|error| format!("{context}: expected a face: {error}"))?;
    let (mesh, location) =
        BRepTool::triangulation(&face, mesh_purpose::NONE).map_err(|error| error.to_string())?;
    let mesh = mesh
        .get()
        .ok_or_else(|| format!("{context}: expected a kernel face triangulation"))?;
    require_face_mesh(mesh, context)?;
    let mut triangles = Vec::new();
    for index in 1..=mesh.nb_triangles() {
        let positions =
            triangle_positions(mesh, index, shape.orientation(), location.transformation())?;
        triangles.push(positions);
    }
    Ok(triangles)
}
fn require_face_mesh(mesh: &Triangulation, context: &str) -> Result<(), String> {
    if mesh.nb_triangles() <= 0 {
        return Err(format!(
            "{context} triangles={}: expected nonempty face mesh",
            mesh.nb_triangles()
        ));
    }
    Ok(())
}

pub(super) fn body_triangles(
    source: &Shape,
    options: PrismMeshOptions,
) -> Result<Vec<[[f64; 3]; 3]>, String> {
    use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
    use martensite_tkg3d::top_abs::ShapeEnum;
    validate_options(options)?;
    let mut copier = crate::checked_shape::copy_shape(source, "display body")
        .map_err(|cause| cause.to_string())?;
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    mesh_shape(&shape, options)?;
    let mut faces = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(&shape, ShapeEnum::Face, &mut faces)
        .map_err(|cause| cause.to_string())?;
    let mut triangles = Vec::new();
    for index in 1..=faces.extent() {
        let face = faces.find_key(index).map_err(|cause| cause.to_string())?;
        triangles.extend(face_triangles(face, "display body face")?);
    }
    if triangles.is_empty() {
        return Err("display body triangles 0: expected a nonempty solid mesh".into());
    }
    Ok(triangles)
}

pub(super) fn named_face_boundaries(
    body: &Shape,
    faces: &[Shape],
    options: PrismMeshOptions,
) -> Result<Vec<[[f64; 3]; 2]>, String> {
    validate_options(options)?;
    let mut copier = crate::checked_shape::copy_shape(body, "face outline")
        .map_err(|cause| cause.to_string())?;
    let mut copied = Vec::new();
    for face in faces {
        let mapped = copier.modified(face).map_err(|cause| cause.to_string())?;
        if mapped.len() != 1 {
            return Err(format!(
                "face outline copies {}: expected one face",
                mapped.len()
            ));
        }
        copied.push(mapped.first().expect("one verified face").clone());
    }
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    mesh_shape(&shape, options)?;
    let body_faces = snapshot::BodyFaceIndex::new(&shape)?;
    let mut result = Vec::new();
    for face in copied {
        let face = body_faces.resolve(&face)?;
        result.extend(boundary_segments(&face_triangles(&face, "face outline")?));
    }
    Ok(result)
}
fn triangle_positions(
    mesh: &Triangulation,
    index: i32,
    orientation: Orientation,
    transform: martensite_tkmath::gp::Transform3,
) -> Result<[[f64; 3]; 3], String> {
    let (a, b, c) = mesh
        .triangle(index)
        .map_err(|error| error.to_string())?
        .get();
    let indices = oriented_indices([a, b, c], orientation)?;
    let mut vertices = [[0.; 3]; 3];
    for (vertex, index) in vertices.iter_mut().zip(indices) {
        *vertex = transformed_node(mesh, index, transform)?;
    }
    Ok(vertices)
}

pub(super) fn oriented_indices(
    [a, b, c]: [i32; 3],
    orientation: Orientation,
) -> Result<[i32; 3], String> {
    let indices = match orientation {
        Orientation::Forward => [a, b, c],
        Orientation::Reversed => [a, c, b],
        other => {
            return Err(format!(
                "prism face orientation {other:?}: expected forward or reversed presentation"
            ));
        }
    };
    Ok(indices)
}
fn transformed_node(
    mesh: &Triangulation,
    index: i32,
    transform: martensite_tkmath::gp::Transform3,
) -> Result<[f64; 3], String> {
    let point = mesh
        .node(index)
        .map_err(|error| error.to_string())?
        .transformed(transform);
    let vertex = [point.x(), point.y(), point.z()];
    if vertex.iter().any(|value| !value.is_finite()) {
        return Err(format!(
            "prism mesh node {index} {vertex:?}: expected finite transformed coordinates"
        ));
    }
    Ok(vertex)
}

#[cfg(test)]
mod boundary_tests {
    use super::*;
    #[test]
    fn a_face_outline_is_the_facet_sides_only_one_facet_uses() {
        let [a, b, c, d] = [[0., 0., 0.], [1., 0., 0.], [1., 1., 0.], [0., 1., 0.]];
        let face = PrismFaceMesh {
            role: PrismFaceRole::Start,
            triangles: vec![[a, b, c], [a, c, d]],
        };
        let outline = face.boundary_segments();
        assert_eq!(outline.len(), 4, "{outline:?}");
        let is_diagonal =
            |[from, to]: &[[f64; 3]; 2]| (*from, *to) == (a, c) || (*from, *to) == (c, a);
        assert!(!outline.iter().any(is_diagonal), "{outline:?}");
    }
}
