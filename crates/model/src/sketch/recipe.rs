// SPDX-License-Identifier: GPL-2.0-only
use super::{
    Sketch2D, SketchArc2D, SketchCircle2D, SketchConstraint, SketchError, SketchLine2D,
    SketchPoint2D,
};
/// Latest planar schema: v2 records order, v3 construction roles, v4 line axis roles, v5 dimension placements, v6 drive settings, v7 analytic ellipses, v8 ground anchors, v9 ellipse minor-radius dimensions, v10 parent and offset splines, v11 offset dimensions, v12 defining-point ownership, v13 point direction dimensions, v14 structural pattern point and radius links, v15 parameter-driven pattern offsets, v16 rectangular pattern records, v17 spline handles, v18 signed offset sources, v19 structural relation owners, v20 branch-preserving corner setbacks, v21 entity formats, v22 authored text, v23 linked text fields.
pub const SKETCH_RECIPE_VERSION: u32 = 23;
/// Owned authored values; no numerical workspace, derived topology or system equations.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRecipe {
    pub version: u32,
    pub points: Vec<SketchPoint2D>,
    pub lines: Vec<SketchLine2D>,
    pub circles: Vec<SketchCircle2D>,
    pub arcs: Vec<SketchArc2D>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub ellipses: Option<Vec<super::SketchEllipse2D>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub splines: Option<Vec<super::SketchSpline2D>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub offset_splines: Option<Vec<super::SketchOffsetSpline2D>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub spline_handles: Option<Vec<super::SketchSplineHandle2D>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub texts: Option<Vec<super::SketchText2D>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub defining_points: Option<Vec<super::SketchEntityId>>,
    pub constraints: Vec<SketchConstraint>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub owned_relations: Option<Vec<super::SketchOwnedRelation>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub pattern_point_links: Option<Vec<super::SketchPatternPointLink>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub pattern_offsets: Option<Vec<super::SketchPatternOffset>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub rectangular_patterns: Option<Vec<super::SketchRectangularPattern>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub pattern_radius_links: Option<Vec<super::SketchPatternRadiusLink>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub entity_order: Option<Vec<super::SketchEntityId>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub construction_curves: Option<Vec<super::SketchEntityId>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub center_points: Option<Vec<super::SketchEntityId>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub centerlines: Option<Vec<super::SketchEntityId>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub entity_formats: Option<Vec<super::SketchEntityFormatEntry>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub dimension_placements: Option<Vec<super::SketchDimensionPlacement>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub reference_dimensions: Option<Vec<super::SketchConstraintId>>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub dimension_limits: Option<Vec<super::SketchDimensionLimitRecipe>>,
}
impl Sketch2D {
    /// Capture authored values in stable order; example: `let recipe = sketch.recipe();`.
    pub fn recipe(&self) -> SketchRecipe {
        let mut recipe = self.geometry_recipe();
        self.capture_recipe_schema(&mut recipe);
        self.capture_dimension_settings_schema(&mut recipe);
        self.capture_ellipse_schema(&mut recipe);
        self.capture_ground_schema(&mut recipe);
        self.capture_ellipse_dimension_schema(&mut recipe);
        self.capture_spline_schema(&mut recipe);
        self.capture_point_ownership_schema(&mut recipe);
        self.capture_point_angle_schema(&mut recipe);
        self.capture_pattern_point_links(&mut recipe);
        self.capture_pattern_offsets(&mut recipe);
        self.capture_rectangular_patterns(&mut recipe);
        if !self.spline_handles.is_empty() {
            self.capture_point_ownership_metadata(&mut recipe);
            recipe.version = 17;
        }
        if recipe.constraints.iter().any(|c| {
            matches!(
                c.relation,
                super::PlanarConstraint::OffsetSplineDistance { signed: true, .. }
            )
        }) {
            self.capture_point_ownership_metadata(&mut recipe);
            recipe.version = 18;
        }
        self.capture_owned_relations(&mut recipe);
        if has_corner_setback(&recipe) {
            recipe.version = 20;
        }
        self.capture_entity_formats(&mut recipe);
        self.capture_text_recipe(&mut recipe);
        recipe
    }
    fn capture_point_angle_schema(&self, recipe: &mut SketchRecipe) {
        if !has_point_angle(recipe) {
            return;
        }
        self.capture_point_ownership_metadata(recipe);
        recipe.version = 13;
    }
    fn geometry_recipe(&self) -> SketchRecipe {
        SketchRecipe {
            version: 1,
            points: self.points.clone(),
            lines: self.lines.clone(),
            circles: self.circles.clone(),
            arcs: self.arcs.clone(),
            ellipses: None,
            splines: None,
            offset_splines: None,
            spline_handles: None,
            texts: None,
            defining_points: None,
            constraints: self.constraints.clone(),
            owned_relations: None,
            pattern_point_links: None,
            pattern_offsets: None,
            rectangular_patterns: None,
            pattern_radius_links: None,
            entity_order: self.saved_entity_order(),
            construction_curves: self.saved_construction_curves(),
            center_points: self.saved_center_points(),
            centerlines: self.saved_centerlines(),
            entity_formats: None,
            dimension_placements: self.saved_dimension_placements(),
            reference_dimensions: self.saved_reference_dimensions(),
            dimension_limits: self.saved_dimension_limits(),
        }
    }
    pub(super) fn capture_dimension_schema_metadata(&self, recipe: &mut SketchRecipe) {
        recipe.entity_order = Some(self.entity_order.clone());
        recipe.construction_curves = Some(self.saved_construction_curves().unwrap_or_default());
        recipe.centerlines = Some(self.centerline_ids());
        recipe.dimension_placements = Some(recipe.dimension_placements.take().unwrap_or_default());
        recipe.reference_dimensions = Some(recipe.reference_dimensions.take().unwrap_or_default());
        recipe.dimension_limits = Some(recipe.dimension_limits.take().unwrap_or_default());
    }
    pub(super) fn capture_complete_schema_metadata(&self, recipe: &mut SketchRecipe) {
        self.capture_dimension_schema_metadata(recipe);
        recipe.ellipses = Some(self.ellipses.clone());
        recipe.reference_dimensions = Some(recipe.reference_dimensions.take().unwrap_or_default());
        recipe.dimension_limits = Some(recipe.dimension_limits.take().unwrap_or_default());
    }
    fn capture_recipe_schema(&self, recipe: &mut SketchRecipe) {
        recipe.version = planar_recipe_version(
            recipe.entity_order.is_some(),
            recipe.construction_curves.is_some(),
            recipe.centerlines.is_some(),
        );
        if recipe.dimension_placements.is_none() {
            return;
        }
        recipe.version = 5;
        recipe.entity_order = Some(self.entity_order.clone());
        recipe.construction_curves = Some(self.saved_construction_curves().unwrap_or_default());
        recipe.centerlines = Some(self.centerline_ids());
    }
    /// Restore into a fresh owner after validation; example: `let sketch = Sketch2D::from_recipe(&recipe)?;`.
    pub fn from_recipe(recipe: &SketchRecipe) -> Result<Self, SketchError> {
        require_recipe_version(recipe.version)?;
        if recipe.version < 20 && has_corner_setback(recipe) {
            return Err(SketchError::Definition(format!(
                "corner setback recipe version {}: expected version 20",
                recipe.version
            )));
        }
        require_point_angle_schema(recipe)?;
        super::point_ownership::require_point_ownership_schema(recipe)?;
        super::spline::require_spline_schema(recipe)?;
        let mut sketch = Self::default();
        super::ground_constraint::require_ground_schema(recipe)?;
        sketch.restore_geometry(recipe)?;
        sketch.restore_ellipses(recipe)?;
        sketch.restore_point_ownership(recipe)?;
        sketch.restore_splines(recipe)?;
        sketch.restore_text_recipe(recipe)?;
        sketch.restore_entity_order(recipe)?;
        sketch.restore_construction_curves(recipe)?;
        sketch.restore_centerlines(recipe)?;
        sketch.restore_center_points(recipe)?;
        sketch.restore_entity_formats(recipe)?;
        for constraint in &recipe.constraints {
            sketch.add_constraint(constraint.id, constraint.relation.clone())?;
        }
        sketch.restore_owned_relations(recipe)?;
        sketch.restore_pattern_point_links(recipe)?;
        sketch.restore_pattern_offsets(recipe)?;
        sketch.restore_rectangular_patterns(recipe)?;
        sketch.restore_dimension_placements(recipe)?;
        sketch.restore_dimension_settings(recipe)?;
        Ok(sketch)
    }
    fn restore_geometry(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        for point in &recipe.points {
            self.add_point(point.id, point.position)?;
        }
        for line in &recipe.lines {
            self.add_line(line.id, line.endpoints)?;
        }
        self.restore_circular_geometry(recipe)
    }
    fn restore_circular_geometry(&mut self, recipe: &SketchRecipe) -> Result<(), SketchError> {
        for circle in &recipe.circles {
            self.add_circle(circle.id, circle.center, circle.radius)?;
        }
        for arc in &recipe.arcs {
            self.add_arc(
                arc.id,
                arc.center,
                [arc.start, arc.end],
                arc.counter_clockwise,
            )?;
        }
        Ok(())
    }
}

fn require_recipe_version(version: u32) -> Result<(), SketchError> {
    if !matches!(version, 1..=SKETCH_RECIPE_VERSION) {
        return Err(SketchError::Definition(format!(
            "sketch recipe version {version}: expected 1 through {SKETCH_RECIPE_VERSION}"
        )));
    }
    Ok(())
}

fn planar_recipe_version(ordered: bool, construction: bool, centerline: bool) -> u32 {
    match (ordered, construction, centerline) {
        (_, _, true) => 4,
        (_, true, false) => 3,
        (true, false, false) => 2,
        (false, false, false) => 1,
    }
}

fn has_point_angle(recipe: &SketchRecipe) -> bool {
    recipe
        .constraints
        .iter()
        .any(|row| matches!(row.relation, super::PlanarConstraint::PointAngle { .. }))
}
fn require_point_angle_schema(recipe: &SketchRecipe) -> Result<(), SketchError> {
    if recipe.version < 13 && has_point_angle(recipe) {
        return Err(SketchError::Definition(format!(
            "point angle recipe version {}: expected version 13",
            recipe.version
        )));
    }
    Ok(())
}

fn has_corner_setback(recipe: &SketchRecipe) -> bool {
    recipe
        .constraints
        .iter()
        .any(|row| matches!(row.relation, super::PlanarConstraint::CornerSetback { .. }))
}
