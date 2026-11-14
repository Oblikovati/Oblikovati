// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    FeatureId, FeatureReadKey,
    work_axis::WorkAxisLine,
    work_plane::{WorkPlaneError, WorkPlaneFaceReference, WorkPlaneResolver},
};
use oblikovati_model::identity::SemanticReferenceName;

fn face_reference() -> WorkPlaneFaceReference {
    WorkPlaneFaceReference::new(
        SemanticReferenceName::new(
            "11111111-2222-3333-4444-555555555555".parse().unwrap(),
            9,
            1,
            "sketch-prism",
            "side",
            &[1, 101],
        )
        .unwrap()
        .encode()
        .unwrap(),
    )
    .unwrap()
}
struct NamedFaceResolver {
    reference: WorkPlaneFaceReference,
    height: f64,
}
impl WorkPlaneResolver for NamedFaceResolver {
    fn axis(&self, id: FeatureId) -> Result<WorkAxisLine, WorkPlaneError> {
        Err(WorkPlaneError::Unavailable {
            feature: id,
            expected: "axis",
        })
    }
    fn plane(&self, id: FeatureId) -> Result<SketchPlane, WorkPlaneError> {
        Err(WorkPlaneError::Unavailable {
            feature: id,
            expected: "plane",
        })
    }
    fn point(&self, id: FeatureId) -> Result<[f64; 3], WorkPlaneError> {
        Err(WorkPlaneError::Unavailable {
            feature: id,
            expected: "point",
        })
    }
    fn face(&self, reference: &WorkPlaneFaceReference) -> Result<SketchPlane, WorkPlaneError> {
        if reference != &self.reference {
            return Err(WorkPlaneError::BodyReference("unexpected face key".into()));
        }
        SketchPlane::new(
            [0., 0., self.height],
            [1., 0., 0.],
            [0., 1., 0.],
            Default::default(),
        )
        .map_err(WorkPlaneError::Frame)
    }
}
#[test]
fn face_support_round_trips_the_name_and_resolves_current_frames() {
    let reference = face_reference();
    let mut sketches = DocumentSketches::default();
    sketches
        .add_on_face(
            DocumentSketchId(1),
            "Face sketch",
            reference.clone(),
            Sketch2D::default(),
        )
        .unwrap();
    let placed = sketches.get(DocumentSketchId(1)).unwrap();
    assert!(
        placed
            .read_footprint()
            .keys()
            .contains(&FeatureReadKey::Feature(FeatureId(9)))
    );
    assert!(placed.plane(None).is_err());
    let resolver = |height| NamedFaceResolver {
        reference: reference.clone(),
        height,
    };
    assert_eq!(
        placed.plane(Some(&resolver(2.))).unwrap().recipe().origin[2],
        2.
    );
    assert_eq!(
        placed.plane(Some(&resolver(3.))).unwrap().recipe().origin[2],
        3.
    );
    let recipe = sketches.recipe();
    assert_eq!(recipe.version, 12);
    let encoded = serde_json::to_string(&recipe).unwrap();
    let restored = DocumentSketches::from_recipe(&serde_json::from_str(&encoded).unwrap()).unwrap();
    assert!(restored.same_recipe(&sketches));
    let mut old = recipe.clone();
    old.version = 11;
    assert!(DocumentSketches::from_recipe(&old).is_err());
    let mut ambiguous = recipe;
    ambiguous.sketches[0].plane_feature = Some(FeatureId(4));
    assert!(DocumentSketches::from_recipe(&ambiguous).is_err());
}
