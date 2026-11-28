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
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let snapshot = session.snapshot();
        let mut query = oblikovati_automation::FileQueryHost::new(&self.files)
            .with_sketch_edit(self.sketch_inspection_edit());
        if oblikovati_automation::requires_part_replay(snapshot) {
            let resolver = self
                .part_scenes
                .sketch_continuity_resolver(&self.files, document)?;
            query = query.with_part_sketch_continuity(resolver);
        }
        populate_browser(&mut browser, snapshot, &Client::new(query))?;
        for feature in snapshot.work_geometry.features() {
            use oblikovati_model::feature::part::PartFeatureDefinition;
            let (length, edge_count, chamfer) = match feature.definition() {
                PartFeatureDefinition::Fillet(definition) => {
                    (definition.radius(), definition.edges().len(), false)
                }
                PartFeatureDefinition::Chamfer(definition) => {
                    (definition.distance(), definition.edges().len(), true)
                }
                _ => continue,
            };
            let radius = match length {
                oblikovati_model::feature::work_geometry::DatumLength::Constant(value) => {
                    format!("{value} cm")
                }
                oblikovati_model::feature::work_geometry::DatumLength::Parameter(id) => {
                    let Some(parameter) = snapshot.parameters.get(id) else {
                        continue;
                    };
                    parameter.name.clone()
                }
            };
            let seed = oblikovati_gui::GuiFilletEditSeed {
                generation: session.generation(),
                radius,
                edge_count,
            };
            if chamfer {
                browser.chamfer_edits.insert(feature.id(), seed);
            } else {
                browser.fillet_edits.insert(feature.id(), seed);
            }
        }
        browser.fillet_v2_edits =
            super::owner_blend::owner_fillet_v2::edit_seeds(snapshot, session.generation());
        browser.chamfer_v2_edits =
            super::owner_blend::owner_chamfer_v2::edit_seeds(snapshot, session.generation());
        browser.shell_v2_edits =
            super::owner_blend::owner_shell_v2::edit_seeds(snapshot, session.generation());
        if let Ok(identity) = self.files.workspace().identity(document) {
            browser.thread_edits = super::owner_blend::owner_thread::edit_seeds(
                snapshot,
                self.part_scenes.current(session).ok(),
                session.history(),
                identity.internal_name,
            );
        }
        browser.split_edits = super::owner_blend::owner_split::edit_seeds(
            snapshot,
            self.part_scenes.current(session).ok(),
            session.history(),
            self.files
                .workspace()
                .identity(document)
                .map_err(|cause| cause.to_string())?
                .internal_name,
        );
        browser.occurrences = occurrences::browser_occurrences(&self.files, document)?;
        browser.selected_occurrence =
            self.selected_occurrence.as_ref().and_then(|(owner, path)| {
                (*owner == document && occurrences::contains_occurrence(&browser.occurrences, path))
                    .then(|| path.clone())
            });
        browser.selected_sketches = self.selected_browser_sketches(document);
        (browser.bodies, browser.surface_bodies) = self.gui_browser_bodies(document);
        if matches!(
            browser.kind,
            oblikovati_gui::GuiBrowserKind::Part | oblikovati_gui::GuiBrowserKind::SheetMetal
        ) {
            browser.materials = self.gui_material_choices(document)?;
            browser.part_material = snapshot
                .assets
                .assignments
                .part_material_id()
                .map(str::to_owned);
            browser.part_material_appearance =
                self.gui_part_material_appearance(&snapshot.assets)?;
            browser.appearances =
                self.gui_appearance_choices(document, browser.part_material_appearance.as_deref())?;
            browser.part_appearance = snapshot
                .assets
                .assignments
                .part_appearance_id()
                .map(str::to_owned);
        } else if !browser.bodies.is_empty() {
            browser.appearances = self.gui_appearance_choices(document, None)?;
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
    browser.part_design_views = snapshot.part_design_views.clone();
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
    browser.hidden_datums = snapshot
        .work_geometry
        .features()
        .iter()
        .filter(|feature| !feature.is_visible())
        .map(|feature| feature.id())
        .collect();
    browser.loft_edits = super::loft::loft_edits(snapshot);
    browser.sweep_edits = super::owner_blend::owner_sweep::edit_seeds(snapshot);
    browser.revolve_edits = super::owner_blend::owner_revolve::edit_seeds(snapshot);
    browser.coil_edits = super::owner_blend::owner_coil::edit_seeds(snapshot);
    browser.emboss_edits = super::owner_blend::owner_emboss::edit_seeds(snapshot);
    browser.derive_edits = super::owner_blend::owner_derive::edit_seeds(snapshot);
    browser.rib_edits = super::owner_blend::owner_rib::edit_seeds(snapshot);
    browser.decal_edits = super::owner_blend::owner_decal::edit_seeds(snapshot);
    browser.import_edits = super::owner_blend::owner_import::edit_seeds(snapshot);
    browser.unwrap_edits = super::owner_blend::owner_unwrap::edit_seeds(snapshot);
    browser.hole_edits = super::owner_blend::owner_hole::edit_seeds(snapshot);
    browser.slot_edits = super::owner_blend::owner_slot::edit_seeds(snapshot);
    browser.combine_edits = super::owner_blend::owner_combine::edit_seeds(snapshot);
    browser.direct_edit_edits = super::owner_blend::owner_direct_edit::edit_seeds(snapshot);
    browser.draft_edits = super::owner_blend::owner_draft::edit_seeds(snapshot);
    browser.delete_face_edits = super::owner_blend::owner_delete_face::edit_seeds(snapshot);
    browser.thicken_offset_edits = super::owner_blend::owner_thicken_offset::edit_seeds(snapshot);
    browser.mark_edits = super::owner_blend::owner_mark::edit_seeds(snapshot);
    browser.third_party_entries = super::owner_blend::owner_import::third_party_entries(snapshot);
    browser.extrusion_edits = snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let oblikovati_model::feature::part::PartFeatureDefinition::SketchPrism(definition) =
                feature.definition()
            else {
                return None;
            };
            if definition.extent().is_some() {
                return None;
            }
            let two_sided = definition.second_depth().is_some();
            let expression =
                |source| prism_distance_expression(snapshot, feature, source, two_sided);
            let distance = expression(definition.displacement()[2])?;
            let second_distance = match definition.second_depth() {
                Some(source) => Some(expression(source)?),
                None => None,
            };
            Some((
                feature.id(),
                oblikovati_gui::GuiExtrudeEditSeed {
                    sketch: definition.sketch(),
                    profile_count: definition.additional_outputs().len() + 1,
                    distance,
                    second_distance,
                    direction: definition.direction(),
                    taper: prism_taper_expression(snapshot, definition),
                },
            ))
        })
        .collect();
    Ok(())
}
fn prism_taper_expression(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    definition: &oblikovati_model::feature::sketch_prism::SketchPrismDefinition,
) -> Option<String> {
    use oblikovati_model::feature::sketch_prism::SketchPrismDirection;
    use oblikovati_model::feature::work_geometry::DatumLength;
    use oblikovati_model::quantity::{Dimension, Quantity};
    if !matches!(
        definition.direction(),
        SketchPrismDirection::Positive | SketchPrismDirection::Negative
    ) || definition.displacement()[..2] != [DatumLength::Constant(0.); 2]
    {
        return None;
    }
    let parameters = &snapshot.parameters;
    let units = parameters.units();
    let radians = match definition.taper() {
        Some(source) => {
            if let Some(id) = source.parameter_id() {
                return parameters.get(id).map(|parameter| parameter.name.clone());
            }
            source.resolve(parameters).ok()?
        }
        None => 0.,
    };
    let value = units.to_preferred(Quantity::new(radians, Dimension::Angle));
    Some(
        if units.from_preferred(value, Dimension::Angle).value == radians {
            format!("{value} {}", units.preferred_name(Dimension::Angle))
        } else {
            format!("{radians} rad")
        },
    )
}

fn prism_distance_expression(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    feature: &oblikovati_model::feature::AuthoredFeature<
        oblikovati_model::feature::part::PartFeatureDefinition,
    >,
    source: oblikovati_model::feature::work_geometry::DatumLength,
    two_sided: bool,
) -> Option<String> {
    use oblikovati_model::feature::work_geometry::DatumLength;
    match source {
        DatumLength::Constant(value) => {
            use oblikovati_model::quantity::{Dimension, Quantity};
            let units = snapshot.parameters.units();
            let length = units.to_preferred(Quantity::new(value, Dimension::Length));
            let unit = units.preferred_name(Dimension::Length);
            Some(
                if units.from_preferred(length, Dimension::Length).value == value {
                    format!("{length} {unit}")
                } else {
                    format!("{value} cm")
                },
            )
        }
        DatumLength::Parameter(id) => {
            let parameter = snapshot.parameters.get(id)?;
            // A/B swapping needs their expressions, without turning two owned lengths into a cycle.
            if two_sided
                && feature
                    .owned_scalars()
                    .iter()
                    .any(|binding| binding.parameter == id)
            {
                return Some(parameter.source.clone());
            }
            Some(parameter.name.clone())
        }
    }
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
#[path = "owner_browser_kind.rs"]
mod owner_browser_kind;
use owner_browser_kind::feature_kind_name;
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
    let operation = definition.boolean_target().map_or_else(
        || "New Solid".to_owned(),
        |target| target.operation().to_string(),
    );
    if let Some(extent) = definition.extent() {
        use oblikovati_model::feature::sketch_prism::SketchPrismExtent;
        let label = match extent {
            SketchPrismExtent::ThroughAll { .. } => "Through All",
            SketchPrismExtent::ToFace { .. } => "To Face",
            SketchPrismExtent::FromTo { .. } => "From–To",
            SketchPrismExtent::DistanceFromPlane { .. } => "Distance from Plane",
        };
        return Some(format!("{operation}, {label}"));
    }
    let depth = definition.displacement()[2]
        .resolve(&snapshot.parameters)
        .ok()?;
    let shown = snapshot
        .parameters
        .units()
        .display_rounded_expression(Quantity::new(depth.abs(), Dimension::Length))
        .ok()?;
    Some(format!("{operation}, {shown}"))
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
