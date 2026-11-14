// SPDX-License-Identifier: GPL-2.0-only
//! Deterministic sketch states. Scenarios recorded in another application name counts and values
//! from that session; these build the same state through the product's own commands.
use super::super::{ApplicationWorld, diagnostics};
use oblikovati_model::sketch::{
    PlanarConstraint, SketchGeometricConstraintCreation, SketchPointInput, SketchPrimitiveCreation,
};

/// Free segments far from the working view, so the recorded count holds without clutter.
const SEED_ORIGIN: [f64; 2] = [5000., 5000.];
const SEED_STEP: f64 = 40.;
/// A drawn line differs from the empty view within this reach of its centre, in pixels.
const LINE_REACH: i32 = 2;
/// A dot ends inside this many pixels of its center; a cross's arms pass it.
const ARC_PICK_FRACTION: f64 = 0.25;
const CROSS_ARM: i32 = 7;
const MARK_REACH: i32 = 1;
/// The mono font's advance; a label's run starts this far per glyph left of its end.
const LABEL_GLYPH_WIDTH: f32 = 7.;
/// From a label's start into the middle of the field left of it.
const FIELD_INSIDE_LABEL: f32 = 40.;
#[path = "fixtures_observation.rs"]
mod observation;
#[path = "fixtures_panels.rs"]
mod panels;
impl ApplicationWorld {
    /// An empty editable sketch; the view with nothing drawn is kept for later comparison.
    pub fn start_empty_sketch(&mut self) {
        self.add_part();
        self.start_sketch();
        self.remember_sketch_baseline();
    }
    pub fn remember_sketch_baseline(&mut self) {
        // The tool buttons settle on the next frame; the viewport region is what gets compared.
        self.pump(Vec::new());
        let capture = super::super::capture_framebuffer(&mut self.app().window.native);
        self.sketch_baseline = Some(capture);
    }
    /// An editable sketch whose status reads `count` dimensions needed.
    pub fn seed_dimensions_needed(&mut self, count: usize) {
        let current = self.dimensions_needed();
        let missing = count
            .checked_sub(current)
            .unwrap_or_else(|| panic!("seed {count}: expected at least the current {current}"));
        for (index, freedom) in free_entity_plan(missing).into_iter().enumerate() {
            self.seed_free_entity(index, freedom);
        }
        self.pump(Vec::new());
        assert_eq!(self.dimensions_needed(), count, "seeded dimensions needed");
    }
    pub fn dimensions_needed(&mut self) -> usize {
        self.state()
            .dimensions_needed
            .expect("expected an edited sketch")
    }
    /// The status bar is what a person reads; the count must be there, not only in the owner.
    pub fn expect_dimensions_needed(&self, count: usize) {
        self.expect_status(&format!("{count} dimensions needed"));
    }
    pub fn expect_ready(&self) {
        self.expect_status("Ready");
    }
    /// Rendered runs carry no spaces, since a space draws no glyph.
    pub fn expect_status(&self, wanted: &str) {
        assert!(
            self.status_shows(wanted),
            "expected {wanted:?} in status {:?}",
            self.status_text()
        );
    }
    pub fn status_shows(&self, wanted: &str) -> bool {
        self.status_text().contains(&wanted.replace(' ', ""))
    }
    /// The status line's own text, read from its window on the rendered frame.
    pub fn status_text(&self) -> String {
        self.window_text("##status-bar")
    }
    /// Text drawn inside one named window, in draw order.
    /// One Measure row as read; spaces are gone from rendered runs.
    pub fn expect_measure(&self, row: &str) {
        let text = self.window_text("Measure");
        assert!(
            text.contains(&row.replace(' ', "")),
            "expected {row:?} in Measure {text:?}"
        );
    }
    /// Open a tool from the dropdown beside a ribbon caption.
    pub fn choose_from_flyout(&mut self, caption: &str, choice: &str) {
        let caption = self.run(caption).clone();
        let arrow = self
            .frame
            .runs
            .iter()
            .find(|run| {
                run.text == "v"
                    && run.minimum[0] > caption.maximum[0]
                    && (run.center()[1] - caption.center()[1]).abs() < 4.
            })
            .expect("expected a dropdown beside the caption")
            .center()
            .map(|value| value as i32);
        self.click(arrow);
        self.pump(Vec::new());
        self.click_text(choice);
        self.pump(Vec::new());
    }
    pub fn window_text(&self, title: &str) -> String {
        let Some((position, size)) = diagnostics::window_geometry(title) else {
            return String::new();
        };
        let inside = |run: &diagnostics::RenderedRun| {
            let center = run.center();
            (0..2).all(|axis| {
                center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis]
            })
        };
        let runs: Vec<&str> = self
            .frame
            .runs
            .iter()
            .filter(|run| inside(run))
            .map(|run| run.text.as_str())
            .collect();
        runs.join(" ")
    }
    pub fn select_line_tool(&mut self) {
        self.click_text("Line");
    }
    pub fn select_point_tool(&mut self) {
        self.click_text("Point");
        self.pump(Vec::new());
    }
    /// A new sketch on a named origin plane, the view with nothing drawn kept for comparison.
    pub fn start_sketch_on(&mut self, plane: &str) {
        self.add_part();
        self.click_text("Start 2D Sketch");
        self.pump(Vec::new());
        self.click_text(plane);
        self.pump(Vec::new());
        self.remember_sketch_baseline();
    }
    /// A new sketch on any plane, for a fixture that names where its points must measure.
    pub fn start_sketch_on_plane(&mut self, plane: oblikovati_model::sketch::SketchPlane) {
        use oblikovati_model::{document_sketch::SketchPlacement, history::DocumentSketchCreation};
        let document = self.add_part();
        let request = DocumentSketchCreation {
            name: "Sketch1".into(),
            placement: SketchPlacement::Fixed(plane),
            orientation: None,
            project_origin: false,
        };
        self.app()
            .owner
            .create_sketch_for_edit(document, &request)
            .unwrap();
        self.pump(Vec::new());
        self.pump(Vec::new());
        self.remember_sketch_baseline();
    }
    /// What a sketch point draws at this pixel, read from the framebuffer.
    pub fn expect_point_mark(&mut self, at: [i32; 2], wanted: PointMark) {
        let capture = super::super::capture_framebuffer(&mut self.app().window.native);
        let baseline = self
            .sketch_baseline
            .as_ref()
            .expect("expected a remembered sketch baseline");
        let shown = point_mark(&capture, baseline, at);
        assert_eq!(shown, wanted, "point mark at {at:?}");
    }
    pub fn double_click(&mut self, position: [i32; 2]) {
        self.pump(super::super::pointer_click_events(position, 2));
    }
    pub fn undo(&mut self) {
        use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};
        self.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    }
    pub fn redo(&mut self) {
        use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};
        self.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    }
    /// A disabled row draws at reduced alpha; an enabled one at full.
    pub fn run_is_enabled(&self, label: &str) -> bool {
        self.run(label).alpha == u8::MAX
    }
    /// A menu row's check mark, for a label that also names a ribbon button.
    pub fn menu_row_checked(&self, label: &str) -> bool {
        self.checked(label)
    }
    pub fn expect_hud_shows_length_and_angle(&self) {
        let text = self.window_text("##sketch-hud");
        assert!(
            text.contains("mm") && text.contains("deg"),
            "expected a length and an angle in the fields {text:?}"
        );
    }
    pub fn expect_hud_shows_length(&self) {
        let text = self.window_text("##sketch-hud");
        assert!(
            text.contains("mm"),
            "expected a length in the fields {text:?}"
        );
    }
    pub fn expect_hud_value_invalid(&self, value: &str) {
        let run = self.run(value);
        let [red, green, blue] = run.color;
        assert!(
            red > 200 && green < 100 && blue < 100,
            "field {value:?} colour {:?}: expected red",
            run.color
        );
    }
    pub fn status_dimensions_needed(&self) -> usize {
        let text = self.status_text();
        let start = text.find("dimensions").unwrap_or(text.len());
        let leading: String = text[..start]
            .chars()
            .rev()
            .take_while(char::is_ascii_digit)
            .collect::<Vec<_>>()
            .into_iter()
            .rev()
            .collect();
        leading
            .parse()
            .unwrap_or_else(|_| panic!("expected a count in status {text:?}"))
    }
    /// The window pixels of the newest line's ends, for a click or a drawn-pixel check.
    pub fn last_line_pixels(&mut self) -> [[i32; 2]; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let geometry = self.edited_geometry();
        let line = geometry.lines().last().copied().expect("expected a line");
        line.endpoints.map(|point| {
            let at = geometry.point(point).unwrap().position;
            crate::HeadWindow::sketch_pixel(&self.app().window, edit.plane, at).unwrap()
        })
    }
    /// A point this fraction along the newest line, in window pixels.
    pub fn last_line_point(&mut self, fraction: f64) -> [i32; 2] {
        let [from, to] = self.last_line_pixels();
        std::array::from_fn(|axis| {
            from[axis] + (f64::from(to[axis] - from[axis]) * fraction) as i32
        })
    }
    /// Inside the triangle the newest three lines close, in window pixels.
    pub fn chain_interior(&mut self) -> [i32; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let geometry = self.edited_geometry();
        let lines = geometry.lines();
        let chain = &lines[lines.len() - 3..];
        let mut sum = [0.; 2];
        for line in chain {
            let at = geometry.point(line.endpoints[0]).unwrap().position;
            sum = [sum[0] + at[0], sum[1] + at[1]];
        }
        let center = [sum[0] / 3., sum[1] / 3.];
        crate::HeadWindow::sketch_pixel(&self.app().window, edit.plane, center).unwrap()
    }
    pub fn edited_geometry(&mut self) -> oblikovati_model::sketch::Sketch2D {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        self.app()
            .owner
            .documents()
            .workspace()
            .session(edit.document)
            .unwrap()
            .snapshot()
            .sketches
            .get(edit.sketch)
            .unwrap()
            .geometry()
            .clone()
    }
    /// One line's point this fraction along it, in window pixels; lines count in creation order.
    pub fn line_point(&mut self, index: usize, fraction: f64) -> [i32; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let geometry = self.edited_geometry();
        let line = geometry.lines()[index];
        let [from, to] = line
            .endpoints
            .map(|point| geometry.point(point).unwrap().position);
        let at = std::array::from_fn(|axis| from[axis] + (to[axis] - from[axis]) * fraction);
        crate::HeadWindow::sketch_pixel(&self.app().window, edit.plane, at).unwrap()
    }
    /// The first arc a quarter along its turn, in window pixels: its middle is a pick of its own.
    pub fn arc_pixel(&mut self) -> [i32; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let geometry = self.edited_geometry();
        let arc = geometry.arcs()[0];
        let center = geometry.point(arc.center).unwrap().position;
        let angle_of = |point| {
            let at: [f64; 2] = geometry.point(point).unwrap().position;
            (at[1] - center[1]).atan2(at[0] - center[0])
        };
        let radius = {
            let start = geometry.point(arc.start).unwrap().position;
            (start[0] - center[0]).hypot(start[1] - center[1])
        };
        let sweep = (angle_of(arc.end) - angle_of(arc.start)).rem_euclid(std::f64::consts::TAU);
        let sweep = if arc.counter_clockwise {
            sweep
        } else {
            sweep - std::f64::consts::TAU
        };
        let middle = angle_of(arc.start) + sweep * ARC_PICK_FRACTION;
        let at = [
            center[0] + radius * middle.cos(),
            center[1] + radius * middle.sin(),
        ];
        crate::HeadWindow::sketch_pixel(&self.app().window, edit.plane, at).unwrap()
    }
    pub fn sketch_pixel_of(&mut self, at: [f64; 2]) -> [i32; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        crate::HeadWindow::sketch_pixel(&self.app().window, edit.plane, at).unwrap()
    }
    pub fn sketch_point_of(&mut self, pixel: [i32; 2]) -> [f64; 2] {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        crate::HeadWindow::sketch_point(&self.app().window, pixel, edit.plane)
            .unwrap()
            .expect("expected the pixel to reach the sketch plane")
    }
    /// Open a dimension's editor from its annotation, as a double-click at Ready does.
    pub fn open_dimension_label(&mut self, value: &str) {
        let at = self.exact_run_center(value);
        self.double_click(at);
        self.pump(Vec::new());
        self.opened_value = Some(value.to_owned());
    }
    /// The centre of a run reading exactly `text`, for an annotation under the pointer.
    pub fn exact_run_center(&self, text: &str) -> [i32; 2] {
        let run = self
            .frame
            .runs
            .iter()
            .rev()
            .find(|run| run.text == text)
            .unwrap_or_else(|| panic!("expected a run {text:?} in {:?}", self.text()));
        diagnostics::pointer_pixels(run.center())
    }
    /// An annotation is its own run reading exactly this; a ribbon word never matches.
    pub fn annotation_shown(&self, text: &str) -> bool {
        self.frame.runs.iter().any(|run| run.text == text)
    }
    pub fn expect_annotation(&self, text: &str) {
        assert!(
            self.annotation_shown(text),
            "expected annotation {text:?} in {:?}",
            self.text()
        );
    }
    pub fn expect_no_annotation(&self, text: &str) {
        assert!(
            !self.annotation_shown(text),
            "expected no annotation {text:?} in {:?}",
            self.text()
        );
    }
    /// One viewport click, a pointer move then a press and a release at the same place.
    pub fn sketch_click(&mut self, position: [i32; 2]) {
        self.click(position);
    }
    pub fn expect_segment_drawn(&mut self, from: [i32; 2], to: [i32; 2]) {
        let missing = self.segment_samples(from, to, |changed| !changed);
        assert!(
            missing.is_empty(),
            "segment {from:?}-{to:?}: expected drawn pixels at {missing:?}"
        );
    }
    pub fn expect_segment_absent(&mut self, from: [i32; 2], to: [i32; 2]) {
        let present = self.segment_samples(from, to, |changed| changed);
        assert!(
            present.is_empty(),
            "segment {from:?}-{to:?}: expected the empty view at {present:?}"
        );
    }
    /// Samples along the segment's middle, clear of end glyphs and the fields at the pointer.
    fn segment_samples(
        &mut self,
        from: [i32; 2],
        to: [i32; 2],
        wanted: fn(bool) -> bool,
    ) -> Vec<[i32; 2]> {
        let capture = super::super::capture_framebuffer(&mut self.app().window.native);
        let baseline = self
            .sketch_baseline
            .as_ref()
            .expect("expected a remembered sketch baseline");
        (3..=7)
            .map(|step| {
                std::array::from_fn(|axis| from[axis] + (to[axis] - from[axis]) * step / 10)
            })
            .filter(|at: &[i32; 2]| wanted(changed_near(&capture, baseline, *at)))
            .collect()
    }
    fn seed_free_entity(&mut self, index: usize, freedom: usize) {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        let at = SEED_ORIGIN[0] + SEED_STEP * index as f64;
        let creation = seed_creation(index, freedom, at);
        let owner = &mut self.app().owner;
        let mut commands = owner.document_commands(edit.document);
        let report = commands
            .create_sketch_primitive(edit.sketch, &creation)
            .unwrap();
        if freedom == 3 {
            commands
                .create_sketch_geometric_constraint(
                    edit.sketch,
                    &SketchGeometricConstraintCreation::Relation(
                        PlanarConstraint::LineHorizontal {
                            line: report.entity,
                        },
                    ),
                )
                .unwrap();
        }
    }
}
/// What a sketch point's marker looks like on screen.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PointMark {
    Absent,
    Dot,
    Cross,
}
/// A cross has arms reaching past a dot's edge, along the axes.
fn point_mark(
    capture: &oblikovati_viewport::ViewportCapture,
    baseline: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
) -> PointMark {
    // A sprite can land a pixel off the projected point, so each sample reads a small patch.
    let changed =
        |dx: i32, dy: i32| changed_within(capture, baseline, [at[0] + dx, at[1] + dy], MARK_REACH);
    if !changed(0, 0) {
        return PointMark::Absent;
    }
    let cross = changed(CROSS_ARM, 0)
        && changed(-CROSS_ARM, 0)
        && changed(0, CROSS_ARM)
        && changed(0, -CROSS_ARM);
    if cross {
        PointMark::Cross
    } else {
        PointMark::Dot
    }
}
fn changed_near(
    capture: &oblikovati_viewport::ViewportCapture,
    baseline: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
) -> bool {
    changed_within(capture, baseline, at, LINE_REACH)
}
fn changed_within(
    capture: &oblikovati_viewport::ViewportCapture,
    baseline: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
    reach: i32,
) -> bool {
    let width = capture.extent[0] as i32;
    (-reach..=reach).any(|dy| {
        (-reach..=reach).any(|dx| {
            let offset = (((at[1] + dy) * width + at[0] + dx) * 4) as usize;
            capture.rgba[offset..offset + 3] != baseline.rgba[offset..offset + 3]
        })
    })
}
fn seed_creation(index: usize, freedom: usize, at: f64) -> SketchPrimitiveCreation {
    if freedom == 2 {
        return SketchPrimitiveCreation::Point {
            position: [at, SEED_ORIGIN[1]],
        };
    }
    SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::New([at, SEED_ORIGIN[1]]),
            // No two seeds parallel: each direction stays its own free direction.
            SketchPointInput::New([at + 3., SEED_ORIGIN[1] + 1. + index as f64 * 0.37]),
        ],
        construction: false,
        centerline: false,
    }
}
/// Four directions per free line, two per free point, three per horizontal line.
fn free_entity_plan(missing: usize) -> Vec<usize> {
    let mut plan = vec![4; missing / 4];
    match missing % 4 {
        2 => plan.push(2),
        3 => plan.push(3),
        1 => {
            plan.pop();
            plan.extend([3, 2]);
        }
        _ => {}
    }
    plan
}
