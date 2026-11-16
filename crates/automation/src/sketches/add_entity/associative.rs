// SPDX-License-Identifier: GPL-2.0-only
//! Keep named primitive sizes as owned sketch dimensions.
use super::*;
use oblikovati_model::{
    expression::Expression,
    history::SketchDimensionCreation,
    sketch::{
        MeasuredDimensionOperands, SketchDimensionalConstraintCreation, SketchPrimitiveCreation,
    },
};

#[derive(Clone)]
pub(super) enum RadiusBinding {
    Circle(String),
    Major(String),
    Minor(String),
}

pub(super) fn named(snapshot: &DocumentSnapshot, expression: &str) -> Result<bool, Error> {
    let parsed = Expression::parse(expression, snapshot.parameters.units()).map_err(api_error)?;
    Ok(!parsed.references().is_empty())
}

pub(super) fn bindings(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<Vec<RadiusBinding>, Error> {
    let mut result = Vec::new();
    if request.kind == "circle"
        && request.variant != "threePoint"
        && named(snapshot, &request.radius)?
    {
        result.push(RadiusBinding::Circle(request.radius.clone()));
    }
    if request.kind == "ellipse" {
        if named(snapshot, &request.major_radius)? {
            result.push(RadiusBinding::Major(request.major_radius.clone()));
        }
        if named(snapshot, &request.minor_radius)? {
            result.push(RadiusBinding::Minor(request.minor_radius.clone()));
        }
    }
    if request.kind == "offsetSpline" && named(snapshot, &request.radius)? {
        return Err(Error("sketch.addEntity offsetSpline radius: expected a literal until associative offset placement is supported".into()));
    }
    Ok(result)
}

impl RadiusBinding {
    fn dimension(
        &self,
        entity: oblikovati_model::sketch::SketchEntityId,
    ) -> SketchDimensionCreation {
        let (operands, expression) = match self {
            Self::Circle(source) => (MeasuredDimensionOperands::Radius { curve: entity }, source),
            Self::Major(source) => (
                MeasuredDimensionOperands::EllipseRadius { ellipse: entity },
                source,
            ),
            Self::Minor(source) => (
                MeasuredDimensionOperands::EllipseMinorRadius { ellipse: entity },
                source,
            ),
        };
        SketchDimensionCreation {
            operands: SketchDimensionalConstraintCreation::Measured(operands),
            expression: expression.clone(),
            reference: false,
            text_point: None,
        }
    }
}

pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    input: SketchPrimitiveCreation,
    kind: &'static str,
    bindings: Vec<RadiusBinding>,
) -> Result<Vec<u8>, Error> {
    let options = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .sketch_solve_options();
    files
        .commands(document)
        .map_err(api_error)?
        .run("Create associative sketch primitive", |snapshot, ids| {
            let report = snapshot
                .create_sketch_primitive(sketch, &input, ids)
                .map_err(api_error)?;
            for binding in bindings {
                snapshot
                    .create_sketch_dimension(
                        sketch,
                        &binding.dimension(report.entity),
                        options,
                        ids,
                    )
                    .map_err(api_error)?;
            }
            creation_reply(Ok(report), kind)
        })
        .map_err(api_error)
}
