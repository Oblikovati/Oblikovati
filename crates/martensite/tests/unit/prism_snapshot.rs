// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::{brep::BRepTool, topods};
use rustycad_tkmath::poly::mesh_purpose;
fn retained_fixture() -> PrismProbeSnapshot {
    retain_planar_prism(&PrismProbeDefinition {
        outer: PrismProbeLoop {
            edge_keys: vec![1, 2, 3, 4],
            vertices: vec![[0., 0.], [4., 0.], [4., 3.], [0., 3.]],
        },
        holes: Vec::new(),
        displacement: [1., 2., -3.],
    })
    .unwrap()
}
fn require_unmeshed(snapshot: &PrismProbeSnapshot) {
    for (_, shape) in &snapshot.faces {
        let face = topods::face(shape).unwrap();
        let (mesh, _) = BRepTool::triangulation(&face, mesh_purpose::NONE).unwrap();
        assert!(
            mesh.get().is_none(),
            "stored analytic face must remain unmeshed"
        );
    }
}
#[test]
fn presentation_does_not_mesh_or_change_the_retained_brep() {
    let snapshot = retained_fixture();
    require_unmeshed(&snapshot);
    let before = report::checked_report(&snapshot.shape, &snapshot.faces).unwrap();
    snapshot
        .presentation(PrismMeshOptions {
            chord_error: 0.05,
            angular_error: 0.5,
        })
        .unwrap();
    require_unmeshed(&snapshot);
    let after = report::checked_report(&snapshot.shape, &snapshot.faces).unwrap();
    assert_eq!(after, before);
    assert_eq!(snapshot.report(), &after);
}
#[test]
fn detached_prism_copies_each_native_role_and_geometry() {
    let snapshot = retained_fixture();
    let detached = snapshot.detached().unwrap();
    assert!(!detached.shape.is_same(&snapshot.shape));
    for ((source_role, source), (copy_role, copy)) in snapshot.faces.iter().zip(&detached.faces) {
        assert_eq!(source_role, copy_role);
        assert!(!copy.is_same(source));
        let source_surface = BRepTool::surface(&topods::face(source).unwrap()).unwrap();
        let copied_surface = BRepTool::surface(&topods::face(copy).unwrap()).unwrap();
        assert!(!std::ptr::addr_eq(
            source_surface.get().unwrap(),
            copied_surface.get().unwrap()
        ));
    }
    assert_eq!(snapshot.report(), detached.report());
}
#[test]
fn mesh_orientation_reverses_only_the_triangle_winding() {
    use rustycad_tkg3d::top_abs::Orientation;
    assert_eq!(
        mesh::oriented_indices([1, 2, 3], Orientation::Forward).unwrap(),
        [1, 2, 3]
    );
    assert_eq!(
        mesh::oriented_indices([1, 2, 3], Orientation::Reversed).unwrap(),
        [1, 3, 2]
    );
    for orientation in [Orientation::Internal, Orientation::External] {
        assert!(
            mesh::oriented_indices([1, 2, 3], orientation)
                .unwrap_err()
                .contains("expected forward or reversed")
        );
    }
}
