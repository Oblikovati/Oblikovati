// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{
    CircumcircleOptions, PlanarCircumcircle, SketchCircleConstruction, SketchError,
    SolvedCircularSeed,
};
use std::{boxed::Box, cell::RefCell, error::Error as StdError};
#[derive(Default)]
pub(super) struct CardinalCircleConstruction {
    pub(super) fail_verification: bool,
    calls: RefCell<Vec<CircumcircleOptions>>,
    solved: RefCell<Vec<SolvedCircularSeed>>,
}
impl SketchCircleConstruction for CardinalCircleConstruction {
    fn arc_direction(&self, _points: [[f64; 2]; 3]) -> Result<bool, Box<dyn StdError>> {
        Err(Box::new(SketchError::Definition(
            "cardinal fake: expected circumcircle calls only".into(),
        )))
    }

    fn circumcircle(
        &self,
        points: [[f64; 2]; 3],
        options: CircumcircleOptions,
    ) -> Result<PlanarCircumcircle, Box<dyn StdError>> {
        self.calls.borrow_mut().push(options);
        Ok(PlanarCircumcircle {
            center: [0.0; 2],
            radius: points[0][0],
            counter_clockwise: true,
            point_residuals: [0.0; 3],
        })
    }
    fn verify(
        &self,
        _points: [[f64; 2]; 3],
        solved: SolvedCircularSeed,
        _options: CircumcircleOptions,
    ) -> Result<(), Box<dyn StdError>> {
        self.solved.borrow_mut().push(solved);
        if self.fail_verification {
            return Err(Box::new(SketchError::Definition(
                "final circular seed rejected".into(),
            )));
        }
        Ok(())
    }
}
fn request(kind: &str) -> AddSketchEntityArgs {
    AddSketchEntityArgs {
        variant: "threePoint".into(),
        construction: true,
        ccw: false,
        radius: "ignored".into(),
        ..literal_input(kind, &[[2.0, 0.0], [0.0, 2.0], [-2.0, 0.0]])
    }
}
pub(super) fn call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    input: &AddSketchEntityArgs,
    backend: &dyn SketchCircleConstruction,
) -> Result<AddSketchEntityResult, oblikovati_api::Error> {
    Client::new(
        FileHost::new(files, source, "test").with_sketch_construction(backend, Default::default()),
    )
    .sketch()
    .add_entity(input)
}
#[test]
fn typed_three_point_curves_use_the_injected_service_and_owned_primitive_identity_order() {
    for kind in ["circle", "arc"] {
        let (mut files, document, mut source) = fresh();
        let backend = CardinalCircleConstruction::default();
        let reply = call(&mut files, &mut source, &request(kind), &backend).unwrap();
        let expected = if kind == "circle" {
            vec![2]
        } else {
            vec![2, 3, 4]
        };
        assert_eq!(reply.point_i_ds, Some(expected));
        assert_eq!(reply.kind, kind);
        assert!(reply.entity_i_ds.is_none());
        assert!(row(&files, reply.entity_id).construction);
        let geometry = owned_geometry(&files, document).sketches()[0].geometry();
        if kind == "arc" {
            assert!(
                geometry
                    .arc(SketchEntityId(reply.entity_id))
                    .unwrap()
                    .counter_clockwise
            );
        }
        assert_eq!(backend.calls.borrow().len(), 1);
        assert_eq!(backend.solved.borrow().len(), 1);
        let created = rows(&files);
        files.commands(document).unwrap().undo().unwrap();
        assert_eq!(rows(&files).len(), 1);
        files.commands(document).unwrap().redo().unwrap();
        assert_eq!(rows(&files), created);
        reopen_with_observer(&mut files, document, &mut source, Default::default());
        assert_eq!(rows(&files), created);
        verify_keys(&files, &created);
    }
}
#[test]
fn final_verification_refusal_rolls_back_geometry_but_retires_new_ids() {
    let (mut files, document, mut source) = fresh();
    let before = rows(&files);
    let backend = CardinalCircleConstruction {
        fail_verification: true,
        ..Default::default()
    };
    expect_api_decline(
        call(&mut files, &mut source, &request("arc"), &backend),
        "final circular seed rejected",
    );
    assert_eq!(backend.solved.borrow().len(), 1);
    assert_eq!(rows(&files), before);
    reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_next_point(&mut files, &mut source, 6);
}
#[test]
fn three_point_service_and_count_are_required_before_reservations() {
    let (mut files, document, mut source) = fresh();
    assert_refused(
        &mut files,
        document,
        &mut source,
        &request("arc"),
        "injected",
    );
    let mut malformed = request("arc");
    malformed.points.as_mut().unwrap().pop();
    assert_refused(&mut files, document, &mut source, &malformed, "exactly 3");
    assert_next_point(&mut files, &mut source, 2);
}
#[test]
fn three_point_literals_and_acceptance_bound_use_the_same_working_scale() {
    let (mut files, _, mut source) = configured_scaled(0.001);
    create_empty_sketch(&mut files, &mut source);
    let backend = CardinalCircleConstruction::default();
    let reply = call(&mut files, &mut source, &request("circle"), &backend).unwrap();
    assert_eq!(row(&files, reply.entity_id).radius, 2.0);
    assert_eq!(backend.solved.borrow()[0].radius, 2000.0);
    assert_eq!(
        backend.calls.borrow()[0].maximum_point_residual,
        1e-9 / 0.001
    );
}
#[test]
fn three_point_expressions_override_literals_in_document_working_units() {
    let (mut files, _, mut source) = configured_scaled(0.001);
    create_empty_sketch(&mut files, &mut source);
    let backend = CardinalCircleConstruction::default();
    let mut input = request("arc");
    input.points = Some(vec![None]);
    input.point_exprs = Some(
        [["20 mm", "0 cm"], ["0 cm", "20 mm"], ["-20 mm", "0 cm"]]
            .map(|pair| Some(pair.map(String::from).to_vec()))
            .to_vec(),
    );
    let reply = call(&mut files, &mut source, &input, &backend).unwrap();
    assert_eq!(row(&files, reply.entity_id).radius, 2.0);
    assert_eq!(backend.solved.borrow()[0].radius, 2000.0);
    assert_eq!(
        backend.solved.borrow()[0].arc,
        Some(([[2000.0, 0.0], [-2000.0, 0.0]], true))
    );
}
