// SPDX-License-Identifier: GPL-2.0-only
use super::inputs::slots::{SlotInput, SlotPlacementInput};
use super::*;
use crate::file_host::SketchConstructionAccess;
use oblikovati_model::{
    history::{SketchSlotCreation, SketchSlotCreationReport},
    sketch::{CircumcircleOptions, SketchSlotPlacement, SolvedCircularSeed},
};
#[derive(Clone, Copy)]
struct SlotVerification<'a> {
    access: SketchConstructionAccess<'a>,
    points: [[f64; 2]; 3],
    options: CircumcircleOptions,
    ccw: bool,
}
pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    input: SlotInput,
    access: Option<SketchConstructionAccess<'_>>,
) -> Result<Vec<u8>, Error> {
    let scale = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot()
        .parameters
        .units()
        .working_scale();
    let (placement, verification) = prepare_placement(input.placement, access, scale)?;
    let creation = SketchSlotCreation {
        placement,
        width: input.width,
        construction: input.construction,
    };
    adopt_slot(files, document, sketch, creation, verification)
}
fn adopt_slot<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    creation: SketchSlotCreation,
    verification: Option<SlotVerification<'_>>,
) -> Result<Vec<u8>, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create sketch slot",
            |snapshot, ids| {
                snapshot
                    .create_sketch_slot(sketch, &creation, ids)
                    .map_err(api_error)
            },
            |snapshot, report| verify_reply(snapshot, report, verification),
        )
        .map_err(api_error)
}
fn prepare_placement(
    input: SlotPlacementInput,
    access: Option<SketchConstructionAccess<'_>>,
    scale: f64,
) -> Result<(SketchSlotPlacement, Option<SlotVerification<'_>>), Error> {
    match input {
        SlotPlacementInput::Authored(placement) => Ok((placement, None)),
        SlotPlacementInput::ThreePoints(points) => prepare_three_points(points, access, scale),
    }
}
fn prepare_three_points(
    points: [[f64; 2]; 3],
    access: Option<SketchConstructionAccess<'_>>,
    scale: f64,
) -> Result<(SketchSlotPlacement, Option<SlotVerification<'_>>), Error> {
    let access = access.ok_or_else(|| {
        Error("arcThreePoint slot: expected an injected sketch construction service".into())
    })?;
    let options = access.policy.in_working_units(scale).map_err(api_error)?;
    let circle = access
        .constructor
        .circumcircle(points, options)
        .map_err(api_error)?;
    let placement = SketchSlotPlacement::Arc {
        center: circle.center,
        endpoints: [points[0], points[2]],
        counter_clockwise: circle.counter_clockwise,
    };
    Ok((
        placement,
        Some(SlotVerification {
            access,
            points,
            options,
            ccw: circle.counter_clockwise,
        }),
    ))
}
fn verify_reply(
    snapshot: &DocumentSnapshot,
    report: SketchSlotCreationReport,
    verification: Option<SlotVerification<'_>>,
) -> Result<Vec<u8>, Error> {
    if let Some(verification) = verification {
        verification.verify(snapshot, &report)?;
    }
    encode(&AddSketchEntityResult {
        entity_id: report.members[0].0,
        kind: "slot".into(),
        entity_i_ds: Some(report.members.into_iter().map(|id| id.0).collect()),
        point_i_ds: (!report.exposed_points.is_empty())
            .then(|| report.exposed_points.into_iter().map(|id| id.0).collect()),
        ..Default::default()
    })
}
impl SlotVerification<'_> {
    fn verify(
        self,
        snapshot: &DocumentSnapshot,
        report: &SketchSlotCreationReport,
    ) -> Result<(), Error> {
        let geometry = snapshot
            .sketches
            .get(report.sketch)
            .map_err(api_error)?
            .geometry();
        let endpoints = [
            geometry
                .point(report.points[7])
                .map_err(api_error)?
                .position,
            geometry
                .point(report.points[6])
                .map_err(api_error)?
                .position,
        ];
        for index in [4, 5] {
            let center = geometry
                .point(report.points[index])
                .map_err(api_error)?
                .position;
            let radius = (endpoints[0][0] - center[0]).hypot(endpoints[0][1] - center[1]);
            let solved = SolvedCircularSeed {
                center,
                radius,
                arc: Some((endpoints, self.ccw)),
            };
            self.access
                .constructor
                .verify(self.points, solved, self.options)
                .map_err(api_error)?;
        }
        Ok(())
    }
}
