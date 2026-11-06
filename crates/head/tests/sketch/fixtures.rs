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
    /// The Line button's variants arrow sits immediately right of its caption.
    pub fn open_line_variants(&mut self) {
        let run = self.run("Line").clone();
        let at = [run.maximum[0] as i32 + 34, run.center()[1] as i32];
        self.click(at);
        self.pump(Vec::new());
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
    /// Make `count` dimensions on far-away geometry, so the next one has the recorded name.
    pub fn seed_named_dimensions(&mut self, count: usize) {
        let edit = self.app().owner.active_sketch_edit().unwrap();
        for index in 0..count {
            let at = SEED_ORIGIN[1] + SEED_STEP * 3. + SEED_STEP * index as f64;
            let creation = seed_creation(index, 2, at);
            let owner = &mut self.app().owner;
            let mut commands = owner.document_commands(edit.document);
            let first = commands
                .create_sketch_primitive(edit.sketch, &creation)
                .unwrap();
            let second = commands
                .create_sketch_primitive(
                    edit.sketch,
                    &SketchPrimitiveCreation::Point {
                        position: [at + 7., SEED_ORIGIN[1] + 5.],
                    },
                )
                .unwrap();
            commands
                .create_sketch_dimension(
                    edit.sketch,
                    &oblikovati_model::history::SketchDimensionCreation {
                        operands:
                            oblikovati_model::sketch::SketchDimensionalConstraintCreation::Measured(
                                oblikovati_model::sketch::MeasuredDimensionOperands::Distance {
                                    points: [first.entity, second.entity],
                                    measure: oblikovati_model::sketch::DistanceMeasure::Aligned,
                                },
                            ),
                        expression: "1 mm".into(),
                        reference: false,
                        text_point: None,
                    },
                )
                .unwrap();
        }
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
        run.center().map(|value| value as i32)
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
fn changed_near(
    capture: &oblikovati_viewport::ViewportCapture,
    baseline: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
) -> bool {
    let width = capture.extent[0] as i32;
    (-LINE_REACH..=LINE_REACH).any(|dy| {
        (-LINE_REACH..=LINE_REACH).any(|dx| {
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
