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
    let faces = faces
        .iter()
        .map(|(role, shape)| face_mesh(*role, shape))
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
    let face = martensite_tkbrep::topods::face(shape)
        .map_err(|error| format!("prism {role:?}: expected a face: {error}"))?;
    let (mesh, location) =
        BRepTool::triangulation(&face, mesh_purpose::NONE).map_err(|error| error.to_string())?;
    let mesh = mesh
        .get()
        .ok_or_else(|| format!("prism {role:?}: expected a kernel face triangulation"))?;
    require_face_mesh(mesh, role)?;
    let mut triangles = Vec::new();
    for index in 1..=mesh.nb_triangles() {
        let positions =
            triangle_positions(mesh, index, shape.orientation(), location.transformation())?;
        triangles.push(positions);
    }
    Ok(PrismFaceMesh { role, triangles })
}
fn require_face_mesh(mesh: &Triangulation, role: PrismFaceRole) -> Result<(), String> {
    if mesh.nb_triangles() <= 0 {
        return Err(format!(
            "prism {role:?} triangles={}: expected nonempty face mesh",
            mesh.nb_triangles()
        ));
    }
    Ok(())
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
