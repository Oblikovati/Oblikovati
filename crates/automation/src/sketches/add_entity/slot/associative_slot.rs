// SPDX-License-Identifier: GPL-2.0-only
//! Constrained slots with a parameter-driven width.
use super::*;
use oblikovati_model::{
    history::{SketchPlacementCreationReport, SketchPlacementFieldLock},
    sketch::SketchPlacementRecipe,
};

fn recipe(creation: SketchSlotCreation) -> Result<(SketchPlacementRecipe, bool), Error> {
    let layout = creation
        .placement
        .layout(creation.width)
        .map_err(api_error)?;
    let curved = matches!(
        creation.placement,
        SketchSlotPlacement::Arc { .. } | SketchSlotPlacement::ArcCenterPoint { .. }
    );
    let mut recipe = if curved {
        let ccw = match creation.placement {
            SketchSlotPlacement::Arc {
                counter_clockwise, ..
            } => counter_clockwise,
            SketchSlotPlacement::ArcCenterPoint { sweep, .. } => sweep > 0.,
            _ => return Err(Error("slot placement: expected an arc placement".into())),
        };
        SketchPlacementRecipe::arc_slot(
            layout.points[4],
            [layout.points[7], layout.points[6]],
            creation.width,
            ccw,
        )
        .map_err(api_error)?
    } else {
        SketchPlacementRecipe::straight_slot([layout.points[5], layout.points[4]], creation.width)
            .map_err(api_error)?
    };
    for member in recipe.members.iter_mut().take(4) {
        member.construction = creation.construction;
    }
    Ok((recipe, curved))
}

fn slot_report(
    report: SketchPlacementCreationReport,
    curved: bool,
) -> Result<SketchSlotCreationReport, Error> {
    let expected_points = if curved { 7 } else { 6 };
    if report.points.len() != expected_points || report.members.len() < 4 {
        return Err(Error(format!(
            "slot placement points {}, members {}: expected {expected_points} points and at least four boundary members",
            report.points.len(),
            report.members.len()
        )));
    }
    let points = if curved {
        [1, 4, 2, 3, 0, 0, 6, 5]
            .into_iter()
            .map(|index| report.points[index])
            .collect()
    } else {
        report.points.clone()
    };
    let exposed_points = if curved {
        Vec::new()
    } else {
        report.points[..4].to_vec()
    };
    Ok(SketchSlotCreationReport {
        sketch: report.sketch,
        members: report.members[..4].to_vec(),
        points,
        exposed_points,
    })
}

pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    creation: SketchSlotCreation,
    verification: Option<SlotVerification<'_>>,
    expression: String,
) -> Result<Vec<u8>, Error> {
    let (recipe, curved) = recipe(creation)?;
    let width = recipe
        .fields
        .iter()
        .position(|field| field.label == "Width")
        .ok_or_else(|| Error("slot placement: expected a Width field".into()))?;
    let mut locks = vec![None; recipe.fields.len()];
    locks[width] = Some(SketchPlacementFieldLock {
        expression,
        text_point: None,
    });
    let options = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .sketch_solve_options();
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create associative sketch slot",
            |snapshot, ids| {
                snapshot
                    .create_sketch_placement(sketch, &recipe, &locks, options, ids)
                    .map_err(api_error)
            },
            |snapshot, report| verify_reply(snapshot, slot_report(report, curved)?, verification),
        )
        .map_err(api_error)
}
