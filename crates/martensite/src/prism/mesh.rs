// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use martensite_tkbrep::{brep::BRepTool, topods::Shape};
use martensite_tkg3d::top_abs::Orientation;
use martensite_tkmath::poly::{Triangulation, mesh_purpose};
use martensite_tkmesh::brep_mesh::{BRepMeshDiscretRoot, BRepMeshIncrementalMesh};
use martensite_tktopalgo::brep_lib::BRepLibToolTriangulatedShape;
/// Absolute display approximation limits in working units and radians; these are not modeling tolerances.
#[derive(Clone, Copy, Debug)]
pub struct PrismMeshOptions {
    pub chord_error: f64,
    pub angular_error: f64,
}
/// One oriented display triangle with optional analytical per-vertex unit normals.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DisplayTriangle {
    pub positions: [[f64; 3]; 3],
    pub normals: Option<[[f64; 3]; 3]>,
}
#[derive(Clone, Debug, PartialEq)]
pub struct PrismFaceMesh {
    pub role: PrismFaceRole,
    pub triangles: Vec<[[f64; 3]; 3]>,
    pub normals: Vec<[[f64; 3]; 3]>,
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
    let shaded = face_display_triangles(shape, &format!("prism {role:?}"))?;
    let triangles = shaded.iter().map(|tri| tri.positions).collect();
    let normals = shaded
        .iter()
        .map(|tri| {
            tri.normals
                .unwrap_or_else(|| [flat_unit_normal(tri.positions); 3])
        })
        .collect();
    Ok(PrismFaceMesh {
        role,
        triangles,
        normals,
    })
}
fn face_display_triangles(shape: &Shape, context: &str) -> Result<Vec<DisplayTriangle>, String> {
    let face = martensite_tkbrep::topods::face(shape)
        .map_err(|error| format!("{context}: expected a face: {error}"))?;
    let (mesh_handle, location) =
        BRepTool::triangulation(&face, mesh_purpose::NONE).map_err(|error| error.to_string())?;
    if mesh_handle.get().is_none() {
        return Err(format!("{context}: expected a kernel face triangulation"));
    }
    BRepLibToolTriangulatedShape::compute_normals(&face, &mesh_handle)
        .map_err(|error| format!("{context}: expected face triangulation normals: {error}"))?;
    let mesh = mesh_handle
        .get()
        .ok_or_else(|| format!("{context}: expected a kernel face triangulation"))?;
    require_face_mesh(mesh, context)?;
    let transform = location.transformation();
    let orientation = shape.orientation();
    let mut triangles = Vec::with_capacity(mesh.nb_triangles() as usize);
    for index in 1..=mesh.nb_triangles() {
        triangles.push(triangle_with_normals(mesh, index, orientation, transform)?);
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

pub(super) fn body_face_triangles(
    source: &Shape,
    options: PrismMeshOptions,
) -> Result<Vec<Vec<[[f64; 3]; 3]>>, String> {
    Ok(body_face_display_triangles(source, options)?
        .into_iter()
        .map(|face| face.into_iter().map(|tri| tri.positions).collect())
        .collect())
}

pub(super) fn body_face_display_triangles(
    source: &Shape,
    options: PrismMeshOptions,
) -> Result<Vec<Vec<DisplayTriangle>>, String> {
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
        triangles.push(face_display_triangles(face, "display body face")?);
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
    Ok(named_face_display_triangles(body, faces, options, false)?
        .iter()
        .flat_map(|triangles| {
            let positions: Vec<[[f64; 3]; 3]> = triangles.iter().map(|t| t.positions).collect();
            boundary_segments(&positions)
        })
        .collect())
}

pub(super) fn named_face_display_triangles(
    body: &Shape,
    faces: &[Shape],
    options: PrismMeshOptions,
    complete: bool,
) -> Result<Vec<Vec<DisplayTriangle>>, String> {
    validate_options(options)?;
    let mut copier = crate::checked_shape::copy_shape(body, "named face display")
        .map_err(|cause| cause.to_string())?;
    let mut copied = Vec::new();
    for face in faces {
        let mapped = copier.modified(face).map_err(|cause| cause.to_string())?;
        if mapped.len() != 1 {
            return Err(format!(
                "named face copies {}: expected one face",
                mapped.len()
            ));
        }
        copied.push(mapped.first().expect("one verified face").clone());
    }
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    mesh_shape(&shape, options)?;
    let body_faces = snapshot::BodyFaceIndex::new(&shape)?;
    if complete && faces.len() != body_faces.count() {
        return Err(format!(
            "named display faces {}, body faces {}: expected one name per face",
            faces.len(),
            body_faces.count()
        ));
    }
    let mut indices = std::collections::BTreeSet::new();
    let mut result = Vec::new();
    for face in copied {
        let face = body_faces.resolve(&face)?;
        let index = body_faces.index(&face);
        if !indices.insert(index) {
            return Err(format!(
                "named display face {index}: expected a unique face identity"
            ));
        }
        result.push(face_display_triangles(&face, "named display face")?);
    }
    Ok(result)
}
fn triangle_with_normals(
    mesh: &Triangulation,
    index: i32,
    orientation: Orientation,
    transform: martensite_tkmath::gp::Transform3,
) -> Result<DisplayTriangle, String> {
    let (a, b, c) = mesh
        .triangle(index)
        .map_err(|error| error.to_string())?
        .get();
    let indices = oriented_indices([a, b, c], orientation)?;
    let mut positions = [[0.; 3]; 3];
    for (vertex, node) in positions.iter_mut().zip(indices) {
        *vertex = transformed_node(mesh, node, transform)?;
    }
    let reversed = orientation == Orientation::Reversed;
    let normals = triangle_normals(mesh, indices, reversed, transform, positions);
    Ok(DisplayTriangle {
        positions,
        normals: Some(normals),
    })
}
fn triangle_normals(
    mesh: &Triangulation,
    indices: [i32; 3],
    reversed: bool,
    transform: martensite_tkmath::gp::Transform3,
    positions: [[f64; 3]; 3],
) -> [[f64; 3]; 3] {
    let fallback = flat_unit_normal(positions);
    indices.map(|node| transformed_normal(mesh, node, reversed, transform).unwrap_or(fallback))
}
fn transformed_normal(
    mesh: &Triangulation,
    index: i32,
    reversed: bool,
    transform: martensite_tkmath::gp::Transform3,
) -> Option<[f64; 3]> {
    let mut direction = mesh.normal(index).ok()?;
    if reversed {
        direction.reverse();
    }
    let direction = direction.transformed(transform);
    let normal = [direction.x(), direction.y(), direction.z()];
    let length = (normal[0] * normal[0] + normal[1] * normal[1] + normal[2] * normal[2]).sqrt();
    (length.is_finite() && length > 1e-12).then(|| normal.map(|v| v / length))
}
fn flat_unit_normal([a, b, c]: [[f64; 3]; 3]) -> [f64; 3] {
    let u = [b[0] - a[0], b[1] - a[1], b[2] - a[2]];
    let v = [c[0] - a[0], c[1] - a[1], c[2] - a[2]];
    let n = [
        u[1] * v[2] - u[2] * v[1],
        u[2] * v[0] - u[0] * v[2],
        u[0] * v[1] - u[1] * v[0],
    ];
    let length = (n[0] * n[0] + n[1] * n[1] + n[2] * n[2]).sqrt();
    if length.is_finite() && length > 1e-12 {
        n.map(|value| value / length)
    } else {
        [0., 0., 1.]
    }
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
            normals: Vec::new(),
        };
        let outline = face.boundary_segments();
        assert_eq!(outline.len(), 4, "{outline:?}");
        let is_diagonal =
            |[from, to]: &[[f64; 3]; 2]| (*from, *to) == (a, c) || (*from, *to) == (c, a);
        assert!(!outline.iter().any(is_diagonal), "{outline:?}");
    }
}
