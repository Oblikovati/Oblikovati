// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::file_host::SketchConstructionAccess;
use oblikovati_model::{
    history::SketchPrimitiveCreationReport,
    sketch::{
        CircumcircleOptions, PlanarCircumcircle, SketchPointInput, SketchPrimitiveCreation,
        SolvedCircularSeed,
    },
};
#[derive(Clone, Copy)]
pub(super) struct ThreePointCreation {
    pub points: [[f64; 2]; 3],
    pub arc: bool,
    pub construction: bool,
}
pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    input: ThreePointCreation,
    access: Option<SketchConstructionAccess<'_>>,
) -> Result<Vec<u8>, Error> {
    let access = access.ok_or_else(|| {
        Error("three-point circle/arc: expected an injected sketch construction service".into())
    })?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let options = access
        .policy
        .in_working_units(snapshot.parameters.units().working_scale())
        .map_err(api_error)?;
    let circle = access
        .constructor
        .circumcircle(input.points, options)
        .map_err(api_error)?;
    adopt_circle(files, document, sketch, input, access, options, circle)
}
fn adopt_circle<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    input: ThreePointCreation,
    access: SketchConstructionAccess<'_>,
    options: CircumcircleOptions,
    circle: PlanarCircumcircle,
) -> Result<Vec<u8>, Error> {
    let primitive = input.primitive(circle);
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create three-point curve",
            |snapshot, ids| {
                snapshot
                    .create_sketch_primitive(sketch, &primitive, ids)
                    .map_err(api_error)
            },
            |snapshot, report| verify_reply(snapshot, report, input, access, options),
        )
        .map_err(api_error)
}
impl ThreePointCreation {
    fn primitive(self, circle: PlanarCircumcircle) -> SketchPrimitiveCreation {
        let center = SketchPointInput::New(circle.center);
        if self.arc {
            return SketchPrimitiveCreation::Arc {
                center,
                endpoints: [self.points[0], self.points[2]].map(SketchPointInput::New),
                counter_clockwise: circle.counter_clockwise,
                construction: self.construction,
            };
        }
        SketchPrimitiveCreation::Circle {
            center,
            radius: circle.radius,
            construction: self.construction,
        }
    }
}
fn verify_reply(
    snapshot: &DocumentSnapshot,
    report: SketchPrimitiveCreationReport,
    input: ThreePointCreation,
    access: SketchConstructionAccess<'_>,
    options: CircumcircleOptions,
) -> Result<Vec<u8>, Error> {
    let solved = solved_seed(snapshot, &report, input.arc)?;
    access
        .constructor
        .verify(input.points, solved, options)
        .map_err(api_error)?;
    creation_reply(Ok(report), if input.arc { "arc" } else { "circle" })
}
fn solved_seed(
    snapshot: &DocumentSnapshot,
    report: &SketchPrimitiveCreationReport,
    arc: bool,
) -> Result<SolvedCircularSeed, Error> {
    let geometry = snapshot
        .sketches
        .get(report.sketch)
        .map_err(api_error)?
        .geometry();
    if arc {
        return solved_arc_seed(geometry, report.entity);
    }
    let circle = geometry.circle(report.entity).map_err(api_error)?;
    Ok(SolvedCircularSeed {
        center: geometry.point(circle.center).map_err(api_error)?.position,
        radius: circle.radius,
        arc: None,
    })
}

fn solved_arc_seed(
    geometry: &oblikovati_model::sketch::Sketch2D,
    entity: oblikovati_model::sketch::SketchEntityId,
) -> Result<SolvedCircularSeed, Error> {
    let curve = geometry.arc(entity).map_err(api_error)?;
    let center = geometry.point(curve.center).map_err(api_error)?.position;
    let endpoints = [
        geometry.point(curve.start).map_err(api_error)?.position,
        geometry.point(curve.end).map_err(api_error)?.position,
    ];
    Ok(SolvedCircularSeed {
        center,
        radius: geometry.arc_radius(entity).map_err(api_error)?,
        arc: Some((endpoints, curve.counter_clockwise)),
    })
}
