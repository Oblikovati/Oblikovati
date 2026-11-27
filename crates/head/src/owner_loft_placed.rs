// SPDX-License-Identifier: GPL-2.0-only
//! Map a viewport pick to a private intermediate Loft section and replay it.
use super::*;
use crate::HeadDimensionAnnotation;
use crate::owner::owner_blend::owner_sweep::ray_segment_fraction;
use oblikovati_automation::{loft_definition, replace_loft_section};
use oblikovati_gui::{GuiDimensionLabelRole, GuiLoftPlacedView};
use oblikovati_model::{feature::sketch_loft::ResolvedSketchLoft, sketch::SketchPlane};

#[derive(Clone, Copy)]
pub(super) struct LoftGuide {
    segment: [[f64; 3]; 2],
    x_direction: [f64; 3],
    length: f64,
}

#[derive(Clone, Copy)]
pub(super) struct PlacedView {
    section: SketchLoftPlacedSection,
    guide: LoftGuide,
    radius: f64,
}

#[derive(Default)]
pub(super) struct LoftPlacement {
    saved: Option<SketchLoftPlacedSection>,
    accepted: Option<PlacedView>,
    pub(super) current: Option<PlacedView>,
    pub(super) guide: Option<LoftGuide>,
    pub(super) editing: bool,
}

impl LoftPlacement {
    pub(super) fn saved(view: PlacedView) -> Self {
        Self {
            saved: Some(view.section),
            accepted: Some(view),
            current: Some(view),
            guide: Some(view.guide),
            editing: false,
        }
    }

    pub(super) fn section(&self) -> Option<SketchLoftPlacedSection> {
        self.current.map(|view| view.section)
    }
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn gui_loft_placed(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiLoftPlacedView> {
        let draft = self
            .loft_selection
            .as_ref()
            .filter(|draft| placement_is_active(draft, document, generation))?;
        let view = draft.placement.current?;
        Some(GuiLoftPlacedView {
            section: view.section,
            guide_length_mm: view.guide.length * 10.,
            area_mm2: std::f64::consts::PI * (view.radius * 10.).powi(2),
            editing: draft.placement.editing,
            dirty: Some(view.section) != draft.placement.saved,
        })
    }

    pub(in crate::owner) fn loft_placement_annotations(
        &self,
        document: u64,
        generation: u64,
    ) -> Vec<HeadDimensionAnnotation> {
        let Some(draft) = self
            .loft_selection
            .as_ref()
            .filter(|draft| placement_is_active(draft, document, generation))
        else {
            return Vec::new();
        };
        let Some(guide) = draft.placement.guide else {
            return Vec::new();
        };
        guide
            .annotations(draft.placement.current)
            .unwrap_or_default()
    }

    pub(super) fn pick_placed_section(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<bool, String> {
        let Some(tolerance) = tolerance else {
            return Ok(false);
        };
        let Some(draft) = self
            .loft_selection
            .as_ref()
            .filter(|draft| draft.feature.is_some() && draft.placement.current.is_none())
        else {
            return Ok(false);
        };
        let Some(guide) = draft.placement.guide else {
            return Ok(false);
        };
        let Some(position) = ray_segment_fraction(ray, guide.segment, tolerance) else {
            return Ok(false);
        };
        let Ok(section) = SketchLoftPlacedSection::new(position, 1.) else {
            return Ok(false);
        };
        self.change_placement(document, section, true)
    }

    pub(super) fn edit_placed(&mut self, document: u64) -> Result<bool, String> {
        let Some(draft) = self.placed_draft(document)? else {
            return Ok(false);
        };
        if draft.placement.current.is_none() || draft.placement.editing {
            return Ok(false);
        }
        draft.placement.editing = true;
        Ok(true)
    }

    pub(super) fn apply_placed(
        &mut self,
        document: u64,
        section: SketchLoftPlacedSection,
    ) -> Result<bool, String> {
        if self
            .placed_draft(document)?
            .is_none_or(|draft| !draft.placement.editing)
        {
            return Ok(false);
        }
        self.change_placement(document, section, false)
    }

    pub(super) fn cancel_placed(&mut self, document: u64) -> Result<bool, String> {
        let Some(draft) = self
            .placed_draft(document)?
            .filter(|draft| draft.placement.editing)
        else {
            return Ok(false);
        };
        let accepted = draft.placement.accepted;
        let feature = draft.feature;
        self.clear_loft_preview();
        let preview = accepted
            .zip(feature)
            .map(|(view, id)| self.preview_placed(document, id, view.section));
        if let Some(draft) = &mut self.loft_selection {
            draft.placement.current = accepted;
            draft.placement.editing = false;
            draft.preview = preview;
        }
        Ok(true)
    }

    fn change_placement(
        &mut self,
        document: u64,
        section: SketchLoftPlacedSection,
        editing: bool,
    ) -> Result<bool, String> {
        let Some(feature) = self.placed_draft(document)?.and_then(|draft| draft.feature) else {
            return Ok(false);
        };
        let view = self.resolve_placed_view(document, feature, section)?;
        let preview = self.preview_placed(document, feature, section);
        if let Some(draft) = &mut self.loft_selection {
            draft.placement.current = Some(view);
            draft.placement.editing = editing;
            if !editing && preview.is_ok() {
                draft.placement.accepted = Some(view);
            }
            draft.preview = Some(preview);
        }
        Ok(true)
    }

    pub(super) fn resolve_placed_guide(
        &mut self,
        document: u64,
        feature: FeatureId,
    ) -> Result<LoftGuide, String> {
        LoftGuide::from_input(&self.resolve_placed_input(document, feature, None)?)
    }

    pub(super) fn resolve_placed_view(
        &mut self,
        document: u64,
        feature: FeatureId,
        section: SketchLoftPlacedSection,
    ) -> Result<PlacedView, String> {
        let input = self.resolve_placed_input(document, feature, Some(section))?;
        let circle = input
            .placed_circle()
            .ok_or("Loft expects an intermediate circle.")?;
        Ok(PlacedView {
            section,
            guide: LoftGuide::from_input(&input)?,
            radius: circle.radius,
        })
    }

    fn resolve_placed_input(
        &mut self,
        document: u64,
        feature: FeatureId,
        section: Option<SketchLoftPlacedSection>,
    ) -> Result<ResolvedSketchLoft, String> {
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        loft_definition(snapshot, feature)
            .map_err(|cause| cause.to_string())?
            .clone()
            .with_placed_section(section)?
            .resolve(snapshot, &datums)
    }

    pub(super) fn preview_placed(
        &mut self,
        document: u64,
        feature: FeatureId,
        section: SketchLoftPlacedSection,
    ) -> Result<(), String> {
        self.preview_selected_blend(document, |candidate, _| {
            replace_loft_section(candidate, feature, section)
        })?;
        Ok(())
    }

    pub(super) fn commit_placed(
        &mut self,
        document: u64,
        generation: u64,
        feature: FeatureId,
        section: SketchLoftPlacedSection,
    ) -> Result<bool, String> {
        self.require_browser_document(document)?;
        let current = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        if generation != current {
            return Err("Loft edit belongs to an earlier document generation.".into());
        }
        oblikovati_automation::edit_part_with_checkpoints(
            &mut self.files,
            &mut self.part_scenes,
            document,
            "Edit Loft",
            |candidate, _| replace_loft_section(candidate, feature, section),
        )
        .map_err(|cause| cause.to_string())?;
        self.clear_loft_preview();
        self.loft_selection = None;
        Ok(true)
    }

    fn placed_draft(&mut self, document: u64) -> Result<Option<&mut LoftSelection>, String> {
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        Ok(self
            .loft_selection
            .as_mut()
            .filter(|draft| placement_is_active(draft, document, generation)))
    }
}

fn placement_is_active(draft: &LoftSelection, document: u64, generation: u64) -> bool {
    draft.document == document
        && draft.generation == generation
        && draft.path_mode == GuiLoftPathMode::PlacedSections
}

impl LoftGuide {
    fn from_input(input: &ResolvedSketchLoft) -> Result<Self, String> {
        if input.center_line.is_none() {
            return Err("Placed Sections require a Center Line.".into());
        }
        let [first, last] = &input.sections;
        let length = first
            .center
            .into_iter()
            .zip(last.center)
            .map(|(a, b)| a - b)
            .fold(0.0_f64, f64::hypot);
        if !length.is_finite() || length <= 0. {
            return Err("Loft section centers must be distinct.".into());
        }
        Ok(Self {
            segment: [first.center, last.center],
            x_direction: first.x_direction,
            length,
        })
    }

    fn annotations(
        self,
        placed: Option<PlacedView>,
    ) -> Result<Vec<HeadDimensionAnnotation>, String> {
        let [start, end] = self.segment;
        let y: [f64; 3] = std::array::from_fn(|axis| (end[axis] - start[axis]) / self.length);
        let projection: f64 = self
            .x_direction
            .into_iter()
            .zip(y)
            .map(|(a, b)| a * b)
            .sum();
        let x = std::array::from_fn(|axis| self.x_direction[axis] - projection * y[axis]);
        let x_length = x.into_iter().fold(0.0_f64, f64::hypot);
        let plane = SketchPlane::new(
            start,
            x.map(|value| value / x_length),
            y,
            Default::default(),
        )
        .map_err(|cause| cause.to_string())?;
        let mut labels = vec![annotation(
            plane,
            [0., 0.],
            [[0., 0.], [0., self.length]],
            "Center Line".into(),
        )];
        if let Some(view) = placed {
            let along = self.length * view.section.position;
            let area = std::f64::consts::PI * (view.radius * 10.).powi(2);
            labels.push(annotation(
                plane,
                [0., along],
                [[-view.radius, along], [view.radius, along]],
                format!(
                    "Point1  Position {:.6}  Area {:.3} mm²",
                    view.section.position, area
                ),
            ));
        }
        Ok(labels)
    }
}

fn annotation(
    plane: SketchPlane,
    position: [f64; 2],
    line: [[f64; 2]; 2],
    text: String,
) -> HeadDimensionAnnotation {
    HeadDimensionAnnotation {
        plane,
        position,
        line: Some(line),
        witnesses: None,
        unit: None,
        text,
        role: GuiDimensionLabelRole::Preview,
        edit: None,
    }
}
