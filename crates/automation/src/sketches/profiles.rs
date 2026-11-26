// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{
    PrismResolutionPolicy, SketchRegionMeasure, certified_sketch_region_measures,
};
use oblikovati_model::{
    history::SketchPolylineCreation, quantity::Dimension, sketch::SketchRectangleSeed,
};

pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: SketchArgs,
    policy: Option<PrismResolutionPolicy>,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let profiles = measures(session.snapshot(), sketch, policy)?
        .into_iter()
        .enumerate()
        .map(|(index, region)| ProfileInfo {
            index: index as i64,
            area: region.area,
            closed: true,
            holes: region.holes as i64,
        })
        .collect();
    encode(&ListProfilesResult {
        profiles: Some(profiles),
    })
}

pub(crate) fn rectangle<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SketchRectangleArgs,
    policy: Option<PrismResolutionPolicy>,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let width = length(snapshot, &request.width, "width")?;
    let height = length(snapshot, &request.height, "height")?;
    let points = SketchRectangleSeed::Corners([[0.0, 0.0], [width, height]])
        .vertices()
        .map_err(api_error)?;
    let input = SketchPolylineCreation {
        points: points.to_vec(),
        closed: true,
        construction: false,
    };
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create sketch rectangle",
            |snapshot, ids| {
                snapshot
                    .create_sketch_polyline(sketch, &input, ids)
                    .map_err(api_error)
            },
            |snapshot, _| {
                let profiles = measures(snapshot, sketch, policy)?.len();
                encode(&SketchRectangleResult {
                    sketch_index: request.sketch_index,
                    profiles: profiles as i64,
                })
            },
        )
        .map_err(api_error)
}

fn length(snapshot: &DocumentSnapshot, source: &str, label: &str) -> Result<f64, Error> {
    snapshot
        .parameters
        .evaluate_scalar_expression(source, Dimension::Length)
        .map(|quantity| quantity.value)
        .map_err(|cause| Error(format!("sketch.rectangle {label} {source:?}: {cause}")))
}

fn measures(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    policy: Option<PrismResolutionPolicy>,
) -> Result<Vec<SketchRegionMeasure>, Error> {
    let policy = policy
        .ok_or_else(|| Error("sketch profiles require a certified extrusion policy".into()))?;
    certified_sketch_region_measures(snapshot, sketch, policy).map_err(api_error)
}
