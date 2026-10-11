// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{SPLINE_FIT_CHORD, SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET};
use oblikovati_model::sketch::{SketchOffsetSpline2D, SketchSpline2D, SketchSplineKind};
const SPLINE_WIRE_KINDS: [&str; 5] = [
    "spline",
    "spline",
    "spline",
    "controlPointSpline",
    "offsetSpline",
];

#[test]
fn typed_spline_enumeration_preserves_fit_methods_for_both_owners_after_reopen_and_copy() {
    for owner in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        assert_spline_enumeration(owner);
    }
}

fn assert_spline_enumeration(owner: DocumentType) {
    let (mut files, document, mut source) = configured_files(owner, Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Authored spline recipes", author_splines)
        .unwrap();
    let original = rows(&files);
    assert_fit_rows(&original);
    verify_keys(&files, &original);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(rows(&files), original);
    assert_spline_copy(&mut files, document, &mut source, &original);
}

fn assert_spline_copy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    original: &[SketchEntityInfo],
) {
    activate_copy(
        files,
        document,
        source,
        Path::new("/fixtures/spline-copy.obk"),
    );
    let copied = rows(files);
    assert_fit_rows(&copied);
    assert_copy_identity(original, &copied);
    verify_keys(files, &copied);
}

fn author_splines(
    snapshot: &mut DocumentSnapshot,
    reservations: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let sketch = reservations
        .allocate_sketch()
        .map_err(|cause| cause.to_string())?;
    let ids: Vec<SketchEntityId> = (0..8)
        .map(|_| reservations.allocate_sketch_entity().unwrap())
        .collect();
    let geometry = spline_geometry(ids.try_into().unwrap());
    snapshot
        .sketches
        .add(sketch, "Spline recipes", SketchPlane::xy(), geometry)
        .map_err(|cause| cause.to_string())
}

fn spline_geometry(ids: [SketchEntityId; 8]) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    for (id, position) in ids[..3].iter().zip([[0.0, 0.0], [1.0, 1.0], [2.0, 0.0]]) {
        geometry.add_point(*id, position).unwrap();
    }
    add_fit_parents(&mut geometry, ids);
    geometry
        .add_spline(
            SketchSpline2D::new(ids[6], ids[..3].to_vec(), false, SketchSplineKind::Control)
                .unwrap(),
        )
        .unwrap();
    geometry
        .add_offset_spline(SketchOffsetSpline2D::new(ids[7], ids[3], -2.0).unwrap())
        .unwrap();
    geometry
}

fn add_fit_parents(geometry: &mut Sketch2D, ids: [SketchEntityId; 8]) {
    let methods = [SPLINE_FIT_SMOOTH, SPLINE_FIT_CHORD, SPLINE_FIT_SWEET];
    for (index, method) in methods.into_iter().enumerate() {
        let parent = SketchSpline2D::new(
            ids[index + 3],
            ids[..3].to_vec(),
            index == 2,
            SketchSplineKind::Fit { method },
        )
        .unwrap();
        geometry.add_spline(parent).unwrap();
    }
}

fn assert_fit_rows(entities: &[SketchEntityInfo]) {
    assert_eq!(entities.len(), 8);
    let methods: Vec<&str> = entities.iter().map(|row| row.fit_method.as_str()).collect();
    assert_eq!(methods, ["", "", "", "smooth", "chord", "sweet", "", ""]);
    assert_spline_shapes(entities);
}

fn assert_spline_shapes(entities: &[SketchEntityInfo]) {
    let kinds: Vec<&str> = entities
        .iter()
        .skip(3)
        .map(|row| row.kind.as_str())
        .collect();
    assert_eq!(kinds, SPLINE_WIRE_KINDS);
    assert!(entities[7].points.as_ref().unwrap().is_empty());
    for entity in &entities[3..7] {
        assert_eq!(entity.points, entities[3].points);
    }
}

fn assert_copy_identity(original: &[SketchEntityInfo], copied: &[SketchEntityInfo]) {
    for (first, second) in original.iter().zip(copied) {
        assert_eq!(
            (first.id, first.index, &first.points),
            (second.id, second.index, &second.points)
        );
        assert_ne!(first.reference_key, second.reference_key);
    }
}
