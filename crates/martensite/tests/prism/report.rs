// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::topods::Face;
fn body_faces() -> (Shape, Vec<(PrismFaceRole, Shape)>) {
    let body = crate::build_probe([2., 3., 4.]).unwrap();
    let mut faces = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(&body, ShapeEnum::Face, &mut faces).unwrap();
    let lineage = (1..=faces.extent())
        .map(|index| {
            (
                PrismFaceRole::ProfileEdge(index as u64),
                faces.find_key(index).unwrap().clone(),
            )
        })
        .collect();
    (body, lineage)
}
#[test]
fn incomplete_or_ambiguous_generated_history_cannot_pass_face_coverage() {
    let (body, mut lineage) = body_faces();
    require_face_coverage(&body, &lineage).unwrap();
    let last = lineage.pop().unwrap();
    assert!(
        require_face_coverage(&body, &lineage)
            .unwrap_err()
            .contains("complete face coverage")
    );
    lineage.push(last);
    lineage[1].1 = lineage[0].1.clone();
    assert!(
        require_face_coverage(&body, &lineage)
            .unwrap_err()
            .contains("exactly one construction role")
    );
}
#[test]
fn nonfinite_face_values_and_empty_faces_are_named_refusals() {
    for (area, centroid) in [
        (0., [0.; 3]),
        (f64::INFINITY, [0.; 3]),
        (1., [f64::NAN, 0., 0.]),
    ] {
        assert!(
            require_finite_face(PrismFaceRole::Start, area, centroid)
                .unwrap_err()
                .contains("finite positive")
        );
    }
    let mut face = Face::default();
    rustycad_tkbrep::brep::BRepBuilder::default().make_face(&mut face);
    assert!(face_report(PrismFaceRole::Start, &face).is_err());
}
