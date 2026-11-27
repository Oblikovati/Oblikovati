// SPDX-License-Identifier: GPL-2.0-only
//! Sketch commands, one folder per scenario feature under `features/sketch/`.
pub(crate) mod command_alias;
pub(crate) mod constraint_glyphs;
pub(crate) mod create_block;
pub(crate) mod curvature;
#[path = "dimension-edit/mod.rs"]
pub(crate) mod dimension_edit;
pub(crate) mod inference_scope;
#[path = "inspect.rs"]
pub(crate) mod inspect;
#[path = "line/mod.rs"]
pub(crate) mod line;
pub(crate) mod marking_menu;
#[path = "precise-input/mod.rs"]
pub(crate) mod precise_input;
pub(crate) mod rectangular_pattern;
#[path = "region-properties/mod.rs"]
pub(crate) mod region_properties;
pub(crate) mod text;

pub(crate) mod axes;

/// Draw sketch reading aids and return the relation under the pointer.
pub(crate) fn draw_viewport_overlays(
    ui: &imgui::Ui,
    request: &crate::GuiFrameRequest,
    dimension_positions: &[Option<crate::dimension_labels::LabelPosition>],
) -> Option<crate::GuiConstraintHover> {
    crate::sketch_pick_cue::draw_sketch_pick_cue(ui, request);
    crate::sketch_pick_cue::draw_model_pick_cue(ui, request);
    axes::draw(ui, request);
    crate::dimension_labels::draw_dimension_labels(ui, request, dimension_positions);
    constraint_glyphs::draw_constraint_glyphs(ui, request)
}
pub(crate) mod move_tool;
