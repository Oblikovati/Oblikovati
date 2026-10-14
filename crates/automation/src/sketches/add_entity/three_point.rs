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
    let solved = solved_seed(snapshot, &report)?;
    access
        .constructor
        .verify(input.points, solved, options)
        .map_err(api_error)?;
    creation_reply(Ok(report), if input.arc { "arc" } else { "circle" })
}
fn solved_seed(
    snapshot: &DocumentSnapshot,
    report: &SketchPrimitiveCreationReport,
) -> Result<SolvedCircularSeed, Error> {
    snapshot
        .sketches
        .get(report.sketch)
        .map_err(api_error)?
        .geometry()
        .read_index()
        .circular_seed(report.entity)
        .map_err(api_error)
}
