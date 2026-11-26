// SPDX-License-Identifier: GPL-2.0-only
//! Body Properties: physical values a user asks for, and a body's appearance override.
use super::*;
use oblikovati_api::{
    client::Client,
    contract::Appearance,
    wire::{AssignAppearanceArgs, BodyPhysicalPropertiesArgs},
};
use oblikovati_gui::{GuiAppearanceChoice, GuiBodyPhysical, GuiBrowserBody};
use oblikovati_model::assets::{AppearanceAsset, AssignmentReference};
use oblikovati_session::selection::SelectionClickMode;

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Each solid's browser row, with what the dialog needs of it.
    pub(super) fn gui_browser_bodies(&self, document: u64) -> Vec<GuiBrowserBody> {
        // A part that cannot evaluate says so on the scene; its folder just stays empty.
        let rows = self.part_scenes.body_rows(&self.files, document);
        rows.unwrap_or_default()
            .into_iter()
            .map(|row| GuiBrowserBody {
                appearance: self.body_appearance(document, &row.key),
                physical: self.calculated_physical(document, &row.key),
                visible: self.session_shows_body(document, &row.key),
                faces: row.faces,
                name: row.name,
                feature: row.feature,
                key: row.key,
                index: row.index,
            })
            .collect()
    }
    fn session_shows_body(&self, document: u64, key: &str) -> bool {
        let session = self.files.workspace().session(document);
        session.is_ok_and(|session| session.body_visible(key))
    }
    fn body_appearance(&self, document: u64, key: &str) -> String {
        let session = self.files.workspace().session(document).ok();
        let reference = AssignmentReference::new(key, vec![]).ok();
        session
            .zip(reference)
            .and_then(|(session, reference)| {
                let assignments = &session.snapshot().assets.assignments;
                assignments
                    .body_appearance_id(&reference)
                    .map(str::to_owned)
            })
            .unwrap_or_default()
    }
    /// Values calculated for this model state; a later edit makes them stale, so they go.
    fn calculated_physical(&self, document: u64, key: &str) -> Option<GuiBodyPhysical> {
        let generation = self.files.workspace().session(document).ok()?.generation();
        self.body_physical
            .get(&(document, key.to_owned()))
            .filter(|(calculated, _)| *calculated == generation)
            .map(|(_, physical)| physical.clone())
    }
    pub(super) fn update_body_physical(
        &mut self,
        document: u64,
        body: i64,
    ) -> Result<bool, String> {
        let rows = self.part_scenes.body_rows(&self.files, document)?;
        let index = usize::try_from(body).ok();
        let key = index
            .and_then(|index| rows.iter().find(|row| row.index as usize == index))
            .map(|row| row.key.clone())
            .ok_or_else(|| format!("body index {body}: expected a current solid of the part"))?;
        let properties = Client::new(self.document_host())
            .body()
            .physical_properties(&BodyPhysicalPropertiesArgs {
                body_index: body,
                ..Default::default()
            })
            .map_err(|cause| cause.to_string())?;
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let shown = GuiBodyPhysical {
            mass: format!("{:.3} g", properties.mass_g),
            area: format!("{:.3} mm^2", properties.surface_area_mm2),
            volume: format!("{:.3} mm^3", properties.volume_mm3),
            center: [
                properties.centroid_x_mm,
                properties.centroid_y_mm,
                properties.centroid_z_mm,
            ]
            .map(|value| format!("{value:.3} mm")),
        };
        self.body_physical
            .insert((document, key), (generation, shown));
        Ok(true)
    }
    /// Apply the appearance and reveal it; Cancel never reaches this operation.
    /// Example: `owner.set_body_appearance(document, key, "abs-black")?`.
    pub(super) fn set_body_appearance(
        &mut self,
        document: u64,
        key: &str,
        appearance: &str,
    ) -> Result<bool, String> {
        self.assign_body_appearance(document, key, appearance)?;
        // Selection recolors the whole body (Go head/ui/highlight.go). A successful
        // appearance edit must expose the result required by the Body Properties benchmark.
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .set_browser_sketch_selection(None, SelectionClickMode::Replace, false)
            .map_err(|cause| cause.to_string())?;
        self.model_face = None;
        self.model_edges.clear();
        self.face_appearance_picks.clear();
        Ok(true)
    }
    /// Give a body its own appearance; an empty id clears it, back to the part's.
    pub(super) fn assign_body_appearance(
        &mut self,
        document: u64,
        key: &str,
        appearance: &str,
    ) -> Result<bool, String> {
        if !appearance.is_empty() {
            Client::new(self.document_host())
                .appearances()
                .assign(&AssignAppearanceArgs {
                    scope: "body".into(),
                    key: key.into(),
                    appearance_id: appearance.into(),
                })
                .map_err(|cause| cause.to_string())?;
            return Ok(true);
        }
        let reference = AssignmentReference::new(key, vec![])?;
        self.files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .run("Clear appearance", |snapshot, _| {
                snapshot
                    .assets
                    .assignments
                    .set_body_appearance(&reference, "")
            })
            .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
    /// Every appearance a selector can offer: the document's, the project's, then the shipped.
    pub(super) fn gui_appearance_choices(
        &self,
        document: u64,
        implied: Option<&str>,
    ) -> Result<Vec<GuiAppearanceChoice>, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let project = self.files.active_project();
        let catalog = self.files.builtin_catalog()?;
        let assets = &session.snapshot().assets;
        let assignments = &assets.assignments;
        let used: std::collections::BTreeSet<_> = assignments
            .part_appearance_id()
            .or(implied)
            .into_iter()
            .chain(assignments.body_appearances().map(|(_, id)| id))
            .chain(assignments.face_appearances().map(|(_, id)| id))
            .collect();
        let library = project
            .into_iter()
            .flat_map(|p| p.assets.appearances().ordered());
        Ok(super::part_assets::collect_choices(
            assets
                .appearances()
                .ordered()
                .chain(library)
                .chain(catalog.appearances().ordered()),
            |asset: &AppearanceAsset| {
                let id = asset.id();
                let library = super::part_assets::asset_library(
                    project.is_some_and(|p| p.assets.appearances().get(&id).is_some()),
                    catalog.appearances().get(&id).is_some(),
                );
                GuiAppearanceChoice {
                    category: catalog.appearance_category(&id),
                    document: assets.appearances().get(&id).is_some() || used.contains(id.as_str()),
                    used: used.contains(id.as_str()),
                    library,
                    id,
                    name: asset.display_name(),
                    swatch: Some(super::part_assets::appearance_swatch(asset)),
                    appearance_preview: Some(browser_finish(asset)),
                }
            },
        ))
    }
}

fn browser_finish(asset: &AppearanceAsset) -> oblikovati_gui::GuiAppearancePreview {
    let specular = asset.specular();
    let fuzz = asset.fuzz();
    let uniform = |value: f32| (value.clamp(0., 1.) * 255.).round() as u8;
    oblikovati_gui::GuiAppearancePreview {
        document_owned: asset.source() == oblikovati_api::types::ASSET_DOCUMENT,
        roughness: uniform(specular.roughness),
        highlights: uniform(specular.weight),
        metalness: uniform(asset.base().metalness),
        anisotropy: uniform(specular.roughness_anisotropy),
        fuzz_weight: uniform(fuzz.weight),
        fuzz_color: [fuzz.color.r, fuzz.color.g, fuzz.color.b].map(uniform),
    }
}
