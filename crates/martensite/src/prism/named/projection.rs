// SPDX-License-Identifier: GPL-2.0-only
//! Exact hidden-line output retains analytic edges, never a mesh.
use super::*;
use martensite_tkbrep::{brep_adaptor::BRepAdaptorCurve, topexp::Explorer, topods};
use martensite_tkernel::standard::Handle;
use martensite_tkg3d::{adaptor3d::Adaptor3dCurve, top_abs::ShapeEnum};
use martensite_tkhlr::{
    hlr_algo::Projector,
    hlr_brep::{Algo, HLRToShape, TypeOfResultingEdge},
};
use martensite_tkmath::gp::{Axis2, Direction3, Point3};
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;

/// A right-handed view frame in model units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct DrawingProjectionFrame {
    pub origin: [f64; 3],
    pub normal: [f64; 3],
    pub x_direction: [f64; 3],
}
/// The kernel classification of a projected edge.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum DrawingProjectionCurveKind {
    Sharp,
    Smooth,
    Sewn,
    Outline,
}
/// An exact projected edge, with native geometry kept private.
pub struct DrawingProjectedCurve {
    pub kind: DrawingProjectionCurveKind,
    shape: Shape,
    range: [f64; 2],
    projector: Projector,
    scope: EvaluationScope,
    document: DocumentGuid,
}
/// Visible and hidden curves generated from a detached exact body.
/// Kernel categories can overlap: an edge can be both sharp and an outline.
pub struct DrawingProjection {
    pub visible: Vec<DrawingProjectedCurve>,
    pub hidden: Vec<DrawingProjectedCurve>,
}
impl DrawingProjectedCurve {
    /// Sample this current curve for display; example: `curve.display_parameters(history, guid, 0.01)?` uses centimetres.
    pub fn display_parameters(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        deflection: f64,
    ) -> Result<Vec<f64>, String> {
        self.require_current(history, document)?;
        super::discretize::curve_parameters(&self.shape, self.range, deflection)
    }
    /// Read the edge's native parameter range; example: `curve.parameter_range(history, document)?`.
    pub fn parameter_range(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<[f64; 2], String> {
        self.require_current(history, document)?;
        Ok(self.range)
    }
    /// Evaluate the exact projected curve in its parameter range; example: `curve.point_at(history, document, parameter)?`.
    pub fn point_at(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        parameter: f64,
    ) -> Result<[f64; 2], String> {
        self.require_current(history, document)?;
        self.evaluate(parameter)
    }
    fn require_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<(), String> {
        if self.document != document || !self.scope.is_current(history) {
            return Err(format!(
                "projected curve document {document:?}: expected its current source document and generation"
            ));
        }
        Ok(())
    }
    fn evaluate(&self, parameter: f64) -> Result<[f64; 2], String> {
        if !parameter.is_finite() || parameter < self.range[0] || parameter > self.range[1] {
            return Err(format!(
                "projection parameter {parameter}: expected a finite value in {:?}",
                self.range
            ));
        }
        let curve = adaptor(&self.shape)?;
        let mut point = curve.value(parameter).map_err(|cause| cause.to_string())?;
        // Parallel HLR removes translation for visibility; drawing coordinates retain the full frame.
        point.transform(self.projector.full_transformation());
        let result = [point.x(), point.y()];
        if result.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "projected point {result:?}: expected finite coordinates"
            ));
        }
        Ok(result)
    }
}
impl NamedPrismSnapshot {
    /// Project the current named body through Martensite's exact hidden-line algorithm.
    /// Example: `body.drawing_projection(history, document, body_key, frame)?`.
    pub fn drawing_projection(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        body_key: &[u8],
        frame: DrawingProjectionFrame,
    ) -> Result<DrawingProjection, String> {
        self.body_report(history, document, body_key)?;
        drawing_projection_bodies(history, document, &[self], frame)
    }
}
/// Project a part's current bodies together so that they can hide each other.
/// Example: `drawing_projection_bodies(history, guid, &bodies, frame)?`.
pub fn drawing_projection_bodies(
    history: &DocumentEditHistory,
    document: DocumentGuid,
    bodies: &[&NamedPrismSnapshot],
    frame: DrawingProjectionFrame,
) -> Result<DrawingProjection, String> {
    if bodies.is_empty() {
        return Err("drawing bodies 0: expected at least one current body".into());
    }
    let axis = projection_axis(frame)?;
    let algo = Handle::new(Algo::new().map_err(|cause| cause.to_string())?);
    let engine = algo
        .get()
        .ok_or("drawing projection: expected a live HLR engine")?;
    let mut keys = std::collections::BTreeSet::new();
    for body in bodies {
        let key = body.body_reference_key(history, document)?;
        if !keys.insert(key.to_vec()) {
            return Err(format!("drawing body key {key:?}: expected unique bodies"));
        }
        let mut copier =
            crate::checked_shape::copy_shape(&body.snapshot.shape, "drawing projection copy")
                .map_err(|cause| cause.to_string())?;
        let shape = copier.shape().map_err(|cause| cause.to_string())?;
        engine.add(shape, 0);
    }
    let projector = Projector::with_frame(axis).map_err(|cause| cause.to_string())?;
    engine.set_projector(&projector);
    engine
        .update()
        .map_err(|cause| format!("drawing projection update: {cause}"))?;
    engine
        .hide()
        .map_err(|cause| format!("drawing projection hide: {cause}"))?;
    let output = HLRToShape::new(&algo);
    let visible = classified_curves(&output, &projector, true, history, document)?;
    if visible.is_empty() {
        return Err(
            "drawing projection: expected at least one visible curve for a valid solid".into(),
        );
    }
    let hidden = classified_curves(&output, &projector, false, history, document)?;
    Ok(DrawingProjection { visible, hidden })
}
fn projection_axis(frame: DrawingProjectionFrame) -> Result<Axis2, String> {
    if frame
        .origin
        .iter()
        .chain(&frame.normal)
        .chain(&frame.x_direction)
        .any(|value| !value.is_finite())
    {
        return Err(format!(
            "drawing frame {frame:?}: expected finite coordinates"
        ));
    }
    let direction = |values: [f64; 3]| {
        Direction3::new(values[0], values[1], values[2]).map_err(|cause| cause.to_string())
    };
    Axis2::with_x_direction(
        Point3::new(frame.origin[0], frame.origin[1], frame.origin[2]),
        direction(frame.normal)?,
        direction(frame.x_direction)?,
    )
    .map_err(|cause| {
        format!("drawing frame {frame:?}: expected independent normal and x direction: {cause}")
    })
}
fn classified_curves(
    output: &HLRToShape,
    projector: &Projector,
    visible: bool,
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<Vec<DrawingProjectedCurve>, String> {
    use DrawingProjectionCurveKind::{Outline, Sewn, Sharp, Smooth};
    let kinds = [
        (Sharp, TypeOfResultingEdge::Sharp),
        (Smooth, TypeOfResultingEdge::Rg1Line),
        (Sewn, TypeOfResultingEdge::RgNLine),
        (Outline, TypeOfResultingEdge::OutLine),
    ];
    let mut curves = Vec::new();
    for (kind, native) in kinds {
        let compound = output
            .compound_of_edges(native, visible, true)
            .map_err(|cause| cause.to_string())?;
        curves.extend(projected_curves(
            &compound, kind, projector, history, document,
        )?);
    }
    Ok(curves)
}
fn projected_curves(
    compound: &Shape,
    kind: DrawingProjectionCurveKind,
    projector: &Projector,
    history: &DocumentEditHistory,
    document: DocumentGuid,
) -> Result<Vec<DrawingProjectedCurve>, String> {
    if compound.is_null() {
        return Ok(Vec::new());
    }
    let mut explorer = Explorer::new(compound, ShapeEnum::Edge, ShapeEnum::Shape)
        .map_err(|cause| cause.to_string())?;
    let mut curves = Vec::new();
    while explorer.more() {
        let shape = explorer.current().clone();
        crate::checked_shape::require_geometry(&shape, "projected drawing edge")
            .map_err(|cause| cause.to_string())?;
        let curve = adaptor(&shape)?;
        let range = [
            curve.first_parameter().map_err(|cause| cause.to_string())?,
            curve.last_parameter().map_err(|cause| cause.to_string())?,
        ];
        if range.iter().any(|value| !value.is_finite()) || range[0] > range[1] {
            return Err(format!(
                "projected curve range {range:?}: expected finite ordered bounds"
            ));
        }
        let projected = DrawingProjectedCurve {
            shape,
            range,
            kind,
            projector: *projector,
            scope: history.evaluation_scope(),
            document,
        };
        projected.evaluate(range[0])?;
        projected.evaluate(range[1])?;
        curves.push(projected);
        explorer.next().map_err(|cause| cause.to_string())?;
    }
    Ok(curves)
}
fn adaptor(shape: &Shape) -> Result<BRepAdaptorCurve, String> {
    let edge = topods::edge(shape).map_err(|cause| cause.to_string())?;
    BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())
}
