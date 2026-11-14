// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    types::{
        DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION, DocumentType,
        SUB_TYPE_SHEET_METAL_PART,
    },
    wire::SketchArgs,
};
use oblikovati_gui::{GuiBrowserFeature, GuiBrowserSketch, GuiModelBrowser};
#[path = "owner_browser/occurrences.rs"]
pub(super) mod occurrences;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn gui_browser_snapshot(&self, document: u64) -> Result<GuiModelBrowser, String> {
        let mut browser = self.gui_browser_root(document)?;
        if !browser.has_origin {
            return Ok(browser);
        }
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let mut query = oblikovati_automation::FileQueryHost::new(&self.files)
            .with_sketch_edit(self.sketch_inspection_edit());
        if oblikovati_automation::requires_part_replay(snapshot) {
            let resolver = self
                .part_scenes
                .sketch_continuity_resolver(&self.files, document)?;
            query = query.with_part_sketch_continuity(resolver);
        }
        populate_browser(&mut browser, snapshot, &Client::new(query))?;
        browser.occurrences = occurrences::browser_occurrences(&self.files, document)?;
        browser.selected_occurrence =
            self.selected_occurrence.as_ref().and_then(|(owner, path)| {
                (*owner == document && occurrences::contains_occurrence(&browser.occurrences, path))
                    .then(|| path.clone())
            });
        browser.selected_sketches = self.selected_browser_sketches(document);
        browser.bodies = self.gui_browser_bodies(document);
        if matches!(
            browser.kind,
            oblikovati_gui::GuiBrowserKind::Part | oblikovati_gui::GuiBrowserKind::SheetMetal
        ) {
            browser.appearances = self.gui_appearance_choices(document)?;
            browser.materials = self.gui_material_choices(document)?;
            browser.part_material = snapshot
                .assets
                .assignments
                .part_material_id()
                .map(str::to_owned);
            browser.part_appearance = snapshot
                .assets
                .assignments
                .part_appearance_id()
                .map(str::to_owned);
        } else if !browser.bodies.is_empty() {
            browser.appearances = self.gui_appearance_choices(document)?;
        }
        browser.selected_feature = self
            .files
            .workspace()
            .session(document)
            .ok()
            .and_then(|session| session.selected_browser_feature());
        if let Ok(session) = self.files.workspace().session(document) {
            browser.selected_body = session.selected_browser_body().map(|(_, feature)| feature);
            browser.root_selected = session.browser_root_selected();
        }
        browser.path = self
            .files
            .path(document)
            .ok()
            .and_then(|path| path.to_str().map(str::to_owned));
        Ok(browser)
    }
    fn gui_browser_root(&self, document: u64) -> Result<GuiModelBrowser, String> {
        let info = self
            .files
            .document_info(document)
            .map_err(|cause| cause.to_string())?;
        let kind = self
            .files
            .document_type(document)
            .map_err(|cause| cause.to_string())?;
        Ok(GuiModelBrowser {
            kind: browser_kind(kind, &info.sub_type),
            name: info.name,
            has_origin: matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY),
            ..Default::default()
        })
    }
}
fn browser_kind(kind: DocumentType, sub_type: &str) -> oblikovati_gui::GuiBrowserKind {
    use oblikovati_gui::GuiBrowserKind;
    match kind {
        DOCUMENT_ASSEMBLY => GuiBrowserKind::Assembly,
        DOCUMENT_DRAWING => GuiBrowserKind::Drawing,
        DOCUMENT_PRESENTATION => GuiBrowserKind::Presentation,
        _ if sub_type == SUB_TYPE_SHEET_METAL_PART.0 => GuiBrowserKind::SheetMetal,
        _ => GuiBrowserKind::Part,
    }
}
fn populate_browser<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    browser: &mut GuiModelBrowser,
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    client: &Client<oblikovati_automation::FileQueryHost<'_, O, W, F>>,
) -> Result<(), String> {
    browser.model_states = snapshot.model_states.clone();
    browser.block_definitions = snapshot
        .sketches
        .block_definitions()
        .iter()
        .map(|definition| definition.name().to_owned())
        .collect();
    browser.block_instances = browser_block_instances(snapshot)?;
    browser.rectangular_patterns = browser_rectangular_patterns(snapshot);
    browser.sketches = snapshot
        .sketches
        .sketches()
        .iter()
        .enumerate()
        .map(|(index, sketch)| browser_sketch(snapshot, sketch, index, client))
        .collect();
    browser.features = snapshot
        .work_geometry
        .features()
        .iter()
        .map(|feature| GuiBrowserFeature {
            id: feature.id(),
            name: feature.name().into(),
            suppressed: snapshot
                .feature_suppressed(feature.id())
                .unwrap_or(feature.suppressed()),
            extended: extended_feature_name(snapshot, feature),
            kind: feature_kind_name(feature.definition()),
        })
        .collect();
    Ok(())
}
fn browser_rectangular_patterns(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
) -> Vec<oblikovati_gui::GuiBrowserRectangularPattern> {
    let mut rows = Vec::new();
    for sketch in snapshot.sketches.sketches() {
        for (index, pattern) in sketch.geometry().rectangular_patterns().iter().enumerate() {
            let resolve = |id| {
                snapshot
                    .parameters
                    .get(id)
                    .map(|state| state.source.clone())
                    .ok_or_else(|| {
                        format!("pattern spacing parameter {id}: expected an existing parameter")
                    })
            };
            let spacing = resolve(pattern.definition.spacing[0]).and_then(|first| {
                resolve(pattern.definition.spacing[1]).map(|second| [first, second])
            });
            rows.push(oblikovati_gui::GuiBrowserRectangularPattern {
                sketch: sketch.id(),
                ordinal: index + 1,
                pattern: pattern.clone(),
                spacing,
            });
        }
    }
    rows
}
fn browser_block_instances(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
) -> Result<Vec<oblikovati_gui::GuiBrowserBlockInstance>, String> {
    let mut rows = Vec::new();
    for sketch in snapshot.sketches.sketches() {
        for instance in sketch.block_instances() {
            rows.push(oblikovati_gui::GuiBrowserBlockInstance {
                sketch: sketch.id(),
                id: instance.id(),
                definition: instance.definition().into(),
                ordinal: instance.ordinal().ok_or_else(|| {
                    format!(
                        "block instance {:?}: expected an assigned browser number",
                        instance.id()
                    )
                })?,
            });
        }
    }
    Ok(rows)
}
/// What a search calls this kind of feature.
fn feature_kind_name(
    definition: &oblikovati_model::feature::part::PartFeatureDefinition,
) -> &'static str {
    use oblikovati_model::feature::part::PartFeatureDefinition::{DeleteBody, SketchPrism};
    match definition {
        SketchPrism(_) => "Extrusion",
        DeleteBody(_) => "Delete Body",
        oblikovati_model::feature::part::PartFeatureDefinition::Fixed(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::Offset { .. }
        | oblikovati_model::feature::part::PartFeatureDefinition::ParallelThroughPoint(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::ThroughThreePoints(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::LineThroughPoint(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::NormalToAxis(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::AtAngleAboutAxis(_)
        | oblikovati_model::feature::part::PartFeatureDefinition::ThroughTwoAxes(_) => "Work Plane",
        _ => "Work Feature",
    }
}
/// What a long row adds to a feature's name: its operation and depth.
fn extended_feature_name(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    feature: &oblikovati_model::feature::AuthoredFeature<
        oblikovati_model::feature::part::PartFeatureDefinition,
    >,
) -> Option<String> {
    use oblikovati_model::{
        feature::part::PartFeatureDefinition::SketchPrism,
        quantity::{Dimension, Quantity},
    };
    let SketchPrism(definition) = feature.definition() else {
        return None;
    };
    let depth = definition.displacement()[2]
        .resolve(&snapshot.parameters)
        .ok()?;
    let shown = snapshot
        .parameters
        .units()
        .display_rounded_expression(Quantity::new(depth.abs(), Dimension::Length))
        .ok()?;
    Some(format!("New Solid, {shown}"))
}
fn browser_sketch<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    sketch: &oblikovati_model::document_sketch::PlacedSketch,
    index: usize,
    client: &Client<oblikovati_automation::FileQueryHost<'_, O, W, F>>,
) -> GuiBrowserSketch {
    let inspection = i64::try_from(index)
        .map_err(|_| format!("browser sketch index {index}: expected an i64 index"))
        .and_then(|sketch_index| {
            client
                .sketch()
                .get(&SketchArgs { sketch_index })
                .map_err(|cause| cause.to_string())
        });
    GuiBrowserSketch {
        id: sketch.id(),
        visible: sketch.visible(),
        line_style: oblikovati_gui::GuiSketchLineStyle {
            color: sketch.line_style().color,
            line_type: sketch.line_style().line_type.clone(),
            weight: sketch.line_style().weight,
            scale: sketch.line_style().scale,
        },
        dimensions_shown: !sketch.line_style().dimensions_hidden,
        name: sketch.name().into(),
        inspection,
        consumed_by: exclusive_sketch_consumer(snapshot, sketch.id()),
    }
}
fn exclusive_sketch_consumer(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
) -> Option<oblikovati_model::feature::FeatureId> {
    let mut consumers = snapshot.work_geometry.sketch_consumers(sketch);
    let first = consumers.next()?;
    // Go app/browser.go keeps a sketch at top level when several features share it.
    consumers.next().is_none().then(|| first.id())
}
