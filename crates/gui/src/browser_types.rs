// SPDX-License-Identifier: GPL-2.0-only
//! Browser snapshots and their identity validation.
use oblikovati_model::{document_sketch::DocumentSketchId, feature::FeatureId};

/// The browser tree a document kind shows; each kind names its own branches.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiBrowserKind {
    #[default]
    Part,
    SheetMetal,
    Assembly,
    Drawing,
    Presentation,
}
/// Immutable document structure; only the desktop owner can activate an edit.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiModelBrowser {
    pub name: String,
    pub kind: GuiBrowserKind,
    pub has_origin: bool,
    /// Part members owned by the document, including its active member.
    pub model_states: Option<oblikovati_model::model_state::ModelStateMembers>,
    /// Persisted part views, including the active row and stored display settings.
    pub part_design_views: Option<oblikovati_model::part_design_view::PartDesignViews>,
    pub sketches: Vec<GuiBrowserSketch>,
    /// Definitions and their placed instances, in authored order.
    pub block_definitions: Vec<String>,
    pub block_instances: Vec<GuiBrowserBlockInstance>,
    /// Rectangular patterns owned by each sketch.
    pub rectangular_patterns: Vec<GuiBrowserRectangularPattern>,
    pub features: Vec<GuiBrowserFeature>,
    /// Authored output and path of each saved Loft.
    pub loft_edits: std::collections::BTreeMap<FeatureId, crate::GuiLoftEditSeed>,
    /// Authored options for editable Sweep features.
    pub sweep_edits: std::collections::BTreeMap<FeatureId, crate::GuiSweepEditSeed>,
    /// Authored options for editable Revolve features.
    pub revolve_edits: std::collections::BTreeMap<FeatureId, crate::GuiRevolveEditSeed>,
    /// Authored options for editable Coil features.
    pub coil_edits: std::collections::BTreeMap<FeatureId, crate::GuiCoilEditSeed>,
    /// Authored options for editable Emboss features.
    pub emboss_edits: std::collections::BTreeMap<FeatureId, crate::GuiEmbossEditSeed>,
    /// Authored options for editable Derived Part features.
    pub derive_edits: std::collections::BTreeMap<FeatureId, crate::GuiDeriveEditSeed>,
    /// Authored options for editable Rib features.
    pub rib_edits: std::collections::BTreeMap<FeatureId, crate::GuiRibEditSeed>,
    /// Authored options for editable Decal features.
    pub decal_edits: std::collections::BTreeMap<FeatureId, crate::GuiDecalEditSeed>,
    /// Authored options for editable Imported Model features.
    pub import_edits: std::collections::BTreeMap<FeatureId, crate::GuiImportEditSeed>,
    /// Authored options for editable Unwrap features.
    pub unwrap_edits: std::collections::BTreeMap<FeatureId, crate::GuiUnwrapEditSeed>,
    /// Authored options for editable Hole features.
    pub hole_edits: std::collections::BTreeMap<FeatureId, crate::GuiHoleEditSeed>,
    /// Authored options for editable Slot features.
    pub slot_edits: std::collections::BTreeMap<FeatureId, crate::GuiSlotEditSeed>,
    /// Authored options for editable Combine features.
    pub combine_edits: std::collections::BTreeMap<FeatureId, crate::GuiCombineEditSeed>,
    /// Authored options for editable Draft features.
    pub draft_edits: std::collections::BTreeMap<FeatureId, crate::GuiDraftEditSeed>,
    /// Authored options for editable Thread features.
    pub thread_edits: std::collections::BTreeMap<FeatureId, crate::GuiThreadEditSeed>,
    /// Authored options for editable Delete Face features.
    pub delete_face_edits: std::collections::BTreeMap<FeatureId, crate::GuiDeleteFaceEditSeed>,
    /// Authored options for editable Thicken/Offset features.
    pub thicken_offset_edits:
        std::collections::BTreeMap<FeatureId, crate::GuiThickenOffsetEditSeed>,
    /// Authored operations for editable Direct Edit features.
    pub direct_edit_edits: std::collections::BTreeMap<FeatureId, crate::GuiDirectEditEditSeed>,
    /// Authored options for editable Split features.
    pub split_edits: std::collections::BTreeMap<FeatureId, crate::GuiSplitEditSeed>,
    /// Authored options for editable Mark features.
    pub mark_edits: std::collections::BTreeMap<FeatureId, crate::GuiMarkEditSeed>,
    /// Authored options for editable Finish features.
    pub finish_edits: std::collections::BTreeMap<FeatureId, crate::GuiFinishEditSeed>,
    /// Named imported mesh entries listed under the 3rd Party browser folder.
    pub third_party_entries: Vec<String>,
    /// Editable extrusion distance sources, indexed by their current feature identities.
    pub extrusion_edits: std::collections::BTreeMap<FeatureId, GuiExtrudeEditSeed>,
    /// Authored radius expressions for editable Fillet features.
    pub fillet_edits: std::collections::BTreeMap<FeatureId, GuiFilletEditSeed>,
    /// Full v1/v2 authored Fillet settings for Properties panel hydration.
    pub fillet_v2_edits: std::collections::BTreeMap<FeatureId, crate::GuiFilletV2EditSeed>,
    pub chamfer_v2_edits: std::collections::BTreeMap<FeatureId, crate::GuiChamferV2EditSeed>,
    pub shell_v2_edits: std::collections::BTreeMap<FeatureId, crate::GuiShellV2EditSeed>,
    /// Authored distance expressions for editable Chamfer features.
    pub chamfer_edits: std::collections::BTreeMap<FeatureId, GuiFilletEditSeed>,
    /// Placed components in assembly order, including their reference status.
    pub occurrences: Vec<GuiBrowserOccurrence>,
    /// The selected occurrence path, scoped to the current document generations.
    pub selected_occurrence: Option<GuiOccurrencePath>,
    pub selected_sketches: Vec<DocumentSketchId>,
    /// The feature row a press chose.
    pub selected_feature: Option<FeatureId>,
    /// The body row a press chose, by the feature that made it.
    pub selected_body: Option<FeatureId>,
    /// The document's own row is chosen.
    pub root_selected: bool,
    /// Where the document is saved, for the commands on its own row.
    pub path: Option<String>,
    /// The part's solids, each made by a feature of the program.
    pub bodies: Vec<GuiBrowserBody>,
    /// The part's named open shells, in their browser order.
    pub surface_bodies: Vec<GuiBrowserBody>,
    pub appearances: Vec<crate::GuiAppearanceChoice>,
    /// Available part materials and the assigned part asset identities.
    pub materials: Vec<GuiMaterialChoice>,
    pub part_material: Option<String>,
    pub part_appearance: Option<String>,
    /// Appearance implied by the assigned part material.
    pub part_material_appearance: Option<String>,
}
/// Current authored inputs for a supported extrusion distance draft.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiExtrudeEditSeed {
    pub sketch: DocumentSketchId,
    pub profile_count: usize,
    pub distance: String,
    pub second_distance: Option<String>,
    pub direction: crate::GuiExtrudeDirection,
    pub taper: Option<String>,
}
/// Authored radius and document generation for a Fillet edit.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiFilletEditSeed {
    pub generation: u64,
    pub radius: String,
    pub edge_count: usize,
}
/// A material available to a selector.
pub type GuiMaterialChoice = crate::GuiAssetChoice;
/// A generation-scoped route through assembly owners and occurrence ordinals.
pub type GuiOccurrencePath = Vec<(u64, u64, usize)>;
/// A placed assembly component and the occurrences under its referenced assembly.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiBrowserOccurrence {
    pub path: GuiOccurrencePath,
    pub name: String,
    pub component: String,
    pub suppressed: bool,
    pub virtual_: bool,
    pub deferred: bool,
    pub error: Option<String>,
    pub children: Vec<GuiBrowserOccurrence>,
}
/// Pattern identity, authored settings and spacing source text for the browser editor.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserRectangularPattern {
    pub sketch: DocumentSketchId,
    pub ordinal: usize,
    pub pattern: oblikovati_model::sketch::SketchRectangularPattern,
    pub spacing: Result<[String; 2], String>,
}
/// A placed block's document identity and browser caption.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiBrowserBlockInstance {
    pub sketch: DocumentSketchId,
    pub id: oblikovati_model::sketch::SketchEntityId,
    pub definition: String,
    pub ordinal: u64,
}
/// Stable sketch identity paired with its current canonical API inspection or local decline.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserSketch {
    pub id: DocumentSketchId,
    pub inspection: Result<oblikovati_api::wire::SketchInfo, String>,
    pub name: String,
    pub visible: bool,
    pub consumed_by: Option<FeatureId>,
    pub line_style: crate::GuiSketchLineStyle,
    /// Its dimensions show while the sketch does.
    pub dimensions_shown: bool,
}
/// One solid in the part, named as the document names it; example: `Solid1` from `Extrude1`.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserBody {
    pub name: String,
    pub feature: FeatureId,
    /// The body's stable reference, for what the owner changes on it.
    pub key: String,
    /// Where the part's API counts it.
    pub index: i64,
    pub visible: bool,
    /// The number of exact faces in this body.
    pub faces: usize,
    /// Its own appearance's id; empty when it looks as the part does.
    pub appearance: String,
    pub physical: Option<crate::GuiBodyPhysical>,
}
/// One authored feature in model program order, with its actual suppression state.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiBrowserFeature {
    pub id: FeatureId,
    pub name: String,
    pub suppressed: bool,
    /// What Show Extended Names adds to the row: its operation and depth.
    pub extended: Option<String>,
    /// What the Find dialog calls this kind of feature; example: `Extrusion`.
    pub kind: &'static str,
}
impl GuiModelBrowser {
    pub(crate) fn has_solid(&self, feature: FeatureId) -> bool {
        self.bodies.iter().any(|body| body.feature == feature)
    }

    pub(crate) fn validate(&self) -> Result<(), String> {
        self.validate_labels()?;
        self.validate_identities()?;
        if let Some(states) = &self.model_states {
            states.validate()?;
        }
        if let Some(views) = &self.part_design_views {
            views.validate()?;
        }
        for id in &self.selected_sketches {
            if !self.sketches.iter().any(|row| row.id == *id) {
                return Err(format!(
                    "browser selected sketch {id:?}: expected a current sketch row"
                ));
            }
        }
        for row in &self.sketches {
            row.validate(&self.features)?;
        }
        self.validate_blocks()?;
        self.validate_patterns()?;
        Ok(())
    }
    fn validate_patterns(&self) -> Result<(), String> {
        let mut identities: std::collections::BTreeSet<_> = self
            .block_instances
            .iter()
            .map(|instance| instance.id)
            .collect();
        for row in &self.rectangular_patterns {
            if row.pattern.id.0 == 0
                || row.ordinal == 0
                || !identities.insert(row.pattern.id)
                || !self.sketches.iter().any(|sketch| sketch.id == row.sketch)
            {
                return Err(format!(
                    "browser rectangular pattern {row:?}: expected a unique identity, positive ordinal and existing sketch"
                ));
            }
            let bad_text = match &row.spacing {
                Ok(values) => values.iter().any(|value| value.contains('\0')),
                Err(cause) => cause.contains('\0'),
            };
            if bad_text {
                return Err(format!(
                    "browser rectangular pattern {:?} spacing {:?}: expected text without NUL",
                    row.pattern.id, row.spacing
                ));
            }
        }
        Ok(())
    }
    fn validate_blocks(&self) -> Result<(), String> {
        let mut identities = std::collections::BTreeSet::new();
        for instance in &self.block_instances {
            if instance.id.0 == 0
                || instance.ordinal == 0
                || !identities.insert(instance.id)
                || !self
                    .sketches
                    .iter()
                    .any(|sketch| sketch.id == instance.sketch)
                || !self.block_definitions.contains(&instance.definition)
            {
                return Err(format!(
                    "browser block instance {instance:?}: expected a unique identity, existing sketch and definition, and positive ordinal"
                ));
            }
        }
        Ok(())
    }
    fn validate_labels(&self) -> Result<(), String> {
        for text in std::iter::once(&self.name)
            .chain(self.sketches.iter().map(|row| &row.name))
            .chain(self.features.iter().map(|row| &row.name))
            .chain(self.block_definitions.iter())
            .chain(
                self.model_states
                    .iter()
                    .flat_map(|states| states.list().iter().map(|row| &row.name)),
            )
        {
            if text.contains('\0') {
                return Err(format!("browser label {text:?}: expected text without NUL"));
            }
        }
        Ok(())
    }
    fn validate_identities(&self) -> Result<(), String> {
        let sketches: std::collections::BTreeSet<_> =
            self.sketches.iter().map(|row| row.id.0).collect();
        let features: std::collections::BTreeSet<_> =
            self.features.iter().map(|row| row.id.0).collect();
        if sketches.len() != self.sketches.len()
            || features.len() != self.features.len()
            || sketches.contains(&0)
            || features.contains(&0)
        {
            return Err(format!(
                "browser identities sketches={sketches:?}, features={features:?}: expected unique nonzero IDs within each namespace"
            ));
        }
        Ok(())
    }
}
impl GuiBrowserSketch {
    fn validate(&self, features: &[GuiBrowserFeature]) -> Result<(), String> {
        self.validate_consumer(features)?;
        let tooltip = match &self.inspection {
            Ok(summary) => &summary.plane,
            Err(cause) => cause,
        };
        if tooltip.contains('\0') {
            return Err(format!(
                "browser sketch {:?} inspection {tooltip:?}: expected text without NUL",
                self.id
            ));
        }
        Ok(())
    }
    fn validate_consumer(&self, features: &[GuiBrowserFeature]) -> Result<(), String> {
        if self
            .consumed_by
            .is_some_and(|owner| !features.iter().any(|feature| feature.id == owner))
        {
            return Err(format!(
                "browser sketch {:?}, owner {:?}: expected an owner in the current feature program",
                self.id, self.consumed_by
            ));
        }
        Ok(())
    }
}
