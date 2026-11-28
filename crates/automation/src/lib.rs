// SPDX-License-Identifier: GPL-2.0-only

//! Typed owner-thread queries and transactional edits; native transport is a separate boundary.
pub mod conditional_prism;
mod datums;
mod extrusion_pattern;
pub mod legacy_profile;
mod move_body;
mod part_feature_transaction;
mod part_owner;
pub use move_body::{PreparedMoveBody, prepare_move_body};
mod reference_bytes;
pub use datums::{
    DATUM_COMMAND_METHODS, DATUM_QUERY_METHODS, requires_part_replay, work_plane_reference,
};
mod clients;
pub use clients::CLIENT_COMMAND_METHODS;
mod attachments;
mod attributes;
mod boundary_patch;
mod chamfer;
mod chamfer_v2;
pub use boundary_patch::{
    BoundaryPatchInputs, PreparedBoundaryPatch, build_boundary_patch_definition,
    prepare_boundary_patch_definition, replace_boundary_patch,
};
pub use chamfer::{PreparedChamfer, prepare_chamfer};
pub use chamfer_v2::{
    ChamferV2Inputs, build_chamfer_v2_definition, prepare_chamfer_v2, replace_chamfer,
};
mod coil;
mod combine;
mod command_owner;
mod decal;
mod delete_face;
mod derive;
mod direct_edit;
mod draft;
mod edit;
mod emboss;
mod extend_surface;
mod extrusion;
mod feature_suppression;
mod file_host;
mod fillet;
mod fillet_v2;
mod finish;
mod hole;
mod import;
mod loft;
mod mark;
mod revolve;
mod rib;
mod shell;
mod shell_v2;
mod slot;
mod split;
mod sweep;
mod thicken_offset;
mod thread;
mod unwrap;
pub use coil::{CoilInputs, PreparedCoil, build_coil_definition, prepare_coil_definition};
pub use combine::{
    CombineInputs, PreparedCombine, build_combine_definition, prepare_combine_definition,
    replace_combine,
};
pub use decal::{
    DecalInputs, PreparedDecal, build_decal_definition, prepare_decal_definition, replace_decal,
};
pub use delete_face::{
    DeleteFaceFeatureInputs, PreparedDeleteFace, build_delete_face_definition,
    prepare_delete_face_definition, replace_delete_face,
};
pub use derive::{
    DeriveInputs, PreparedDerivedPart, build_derived_part_definition,
    prepare_derived_part_definition, replace_derived_part,
};
pub use direct_edit::{
    DirectEditFeatureInputs, PreparedDirectEdit, build_direct_edit_definition,
    prepare_direct_edit_definition, replace_direct_edit,
};
pub use draft::{
    DraftFeatureInputs, PreparedDraft, build_draft_definition, prepare_draft_definition,
    replace_draft,
};
pub use emboss::{
    EmbossInputs, PreparedEmboss, build_emboss_definition, prepare_emboss_definition,
};
pub use extend_surface::{
    ExtendSurfaceInputs, PreparedExtendSurface, build_extend_surface_definition,
    prepare_extend_surface_definition, replace_extend_surface,
};
pub use extrusion::{
    CapturedExtrudeCandidate, ExtrudePrefixReplay, ExtrudeRegionAccess, ExtrudeReplayPrefix,
    PreparedExtrusion, commit_prepared_extrusion, evaluated_extrude_distance, prepare_extrusion,
    prepare_owned_extrusion, prepare_owned_extrusion_profiles, validate_extrude_distance,
};
pub use fillet::{
    CertifiedFilletAccess, CertifiedFilletValidation, FilletAccess, FilletValidation,
    PreparedFillet, prepare_fillet,
};
pub use fillet_v2::{
    FilletCornerSetbackInput, FilletEdgeSetInput, FilletV2Inputs, FilletVariablePointInput,
    build_fillet_v2_definition, evaluate_positive_cm, prepare_fillet_v2, replace_fillet,
};
pub use finish::{
    FinishFeatureInputs, PreparedFinish, build_finish_definition, prepare_finish_definition,
    replace_finish,
};
pub use hole::{
    HoleFeatureInputs, HoleSeatInput, HoleTerminationInput, PreparedHole, build_hole_definition,
    prepare_hole, prepare_hole_definition, replace_hole,
};
pub use import::{
    ImportInputs, PreparedImport, build_import_definition, prepare_import_definition,
    replace_imported_model,
};
pub use loft::{
    PreparedLoft, loft_definition, prepare_loft, replace_loft_section, replace_loft_takeoff,
};
pub use mark::{
    MarkFeatureInputs, PreparedMark, build_mark_definition, prepare_mark_definition, replace_mark,
};
pub use revolve::{
    PreparedRevolve, REVOLVE_ANGLE_ROLE, REVOLVE_SECOND_ANGLE_ROLE, RevolveInputs,
    automatic_revolve_profile, canonical_angle_expression, parse_positive_turn_angle,
    prepare_revolve_definition,
};
pub use rib::{
    PreparedRib, RibInputs, build_rib_definition, open_profile_lines, prepare_rib_definition,
    replace_rib,
};
pub use shell::{PreparedShell, prepare_shell};
pub use shell_v2::{
    ShellV2Inputs, ShellV2OverrideInput, build_shell_v2_definition, prepare_shell_v2, replace_shell,
};
pub use slot::{
    PreparedSlot, SlotFeatureInputs, SlotSeatInput, build_slot_definition, prepare_slot_definition,
    replace_slot,
};
pub use split::{
    PreparedSplit, SplitFeatureInputs, build_split_definition, prepare_split_definition,
    replace_split,
};
pub use sweep::{
    PreparedSweep, automatic_sweep, prepare_surface_sweep, prepare_sweep, prepare_sweep_for_target,
    prepare_sweep_taper, prepare_sweep_twist, sweep_boolean_target,
};
pub use thicken_offset::{
    PreparedThickenOffset, ThickenOffsetInputs, build_thicken_offset_definition,
    prepare_thicken_offset_definition, replace_thicken_offset,
};
pub use thread::{
    PreparedThread, ThreadFeatureInputs, build_thread_definition, prepare_thread_definition,
    replace_thread,
};
pub use unwrap::{
    PreparedUnwrap, UnwrapInputs, build_unwrap_definition, prepare_unwrap_definition,
    replace_unwrap,
};
mod interests;
pub use file_host::{
    APPLICATION_PREFERENCE_METHODS, ASSEMBLY_COMMAND_METHODS, AssemblyReferenceFailure,
    AssemblyReferenceReport, DERIVED_COMMAND_METHODS, DOCUMENT_UPDATE_METHODS,
    DocumentObserverFactory, FILE_COMMAND_METHODS, FileHost, creation_package,
};
pub mod dispatch;
mod file_queries;
mod groups_edit;
mod host;
mod model_selection;
mod model_states;
mod part_design_views;
mod part_view_body_visibility;
mod part_view_display;
mod part_view_replay;
mod part_view_transactions;
pub use part_view_display::PartViewDisplayAccess;
mod selection_events;
pub use file_queries::{FILE_QUERY_METHODS, FileQueryHost};
pub use selection_events::{
    EventRelay, EventSink, SelectionClick, SelectionDelete, SelectionDeletionDispatch,
    SelectionDispatch,
};
mod options;
mod settings_edit;
mod sketch_edit_state;
mod window_capture;
pub use window_capture::{WINDOW_CAPTURE_METHODS, WindowCaptureAccess};
mod sketch_inference;
mod sketch_scope;
pub use sketch_edit_state::SketchInspectionEdit;
mod sketches;
pub use edit::EDIT_METHODS;
pub use host::SessionHost;
pub use options::SKETCH_OPTION_METHODS;
pub use sketches::{SKETCH_COMMAND_METHODS, SKETCH_QUERY_METHODS};
mod collection;
mod detail;
pub use detail::parameter_detail;
mod checkpoint_replay;
pub use checkpoint_replay::edit_with_owner as edit_part_with_checkpoints;
mod document_sketch_settings;
mod end_of_part;
mod expressions;
mod feature_lifecycle;
mod model_tree;
mod parameter_choices;
mod parameter_edit;
mod parameter_exchange;
mod parameter_tolerance;
mod parameter_update;
mod parameter_value;
mod parameters;
mod properties;
pub mod ribbon;
mod transactions;
mod units;
mod units_edit;
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::{parameter::ParameterCollection, units::UnitsOfMeasure};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
pub use transactions::TRANSACTION_METHODS;

/// The exact query inventory; version compatibility alone does not imply other methods exist.
pub const QUERY_METHODS: [&str; 41] = [
    METHOD_PART_DESIGN_VIEWS_GET_CAMERA,
    METHOD_PART_DESIGN_VIEWS_LIST,
    METHOD_MODEL_STATES_VERIFY_TABLE,
    METHOD_MODEL_STATES_LIST,
    METHOD_MODEL_STATES_GET_TABLE,
    METHOD_APPLICATION_API_VERSION,
    METHOD_CLIENT_APPS_LIST,
    METHOD_MODEL_SELECTION,
    METHOD_SKETCH_GET_INFERENCE_OPTIONS,
    METHOD_SKETCH_GET_INFERENCE_SCOPE,
    METHOD_DOCUMENTS_LIST_ATTACHMENTS,
    METHOD_DOCUMENTS_LIST_INTERESTS,
    METHOD_DOCUMENTS_HAS_INTEREST,
    METHOD_ATTRIBUTES_GET,
    METHOD_ATTRIBUTES_LIST,
    METHOD_ATTRIBUTES_LIST_SETS,
    METHOD_ATTRIBUTES_FIND,
    METHOD_DOCUMENTS_GET_UNITS,
    METHOD_DOCUMENTS_LIST_PROPERTIES,
    METHOD_DOCUMENTS_GET_PROPERTY,
    METHOD_TRANSACTION_STATE,
    METHOD_TRANSACTION_HISTORY,
    METHOD_PARAMETERS_LIST,
    METHOD_PARAMETERS_GET_DETAIL,
    METHOD_PARAMETERS_GET_SETTINGS,
    METHOD_PARAMETERS_GROUPS_LIST,
    METHOD_PARAMETERS_GET,
    METHOD_PARAMETERS_EXPORT,
    METHOD_PARAMETERS_DRIVEN_BY,
    METHOD_PARAMETERS_DEPENDENTS,
    METHOD_UNITS_GET_VALUE_FROM_EXPRESSION,
    METHOD_UNITS_GET_DATABASE_UNITS_FROM_EXPRESSION,
    METHOD_UNITS_IS_EXPRESSION_VALID,
    METHOD_UNITS_COMPATIBLE_UNITS,
    METHOD_UNITS_GET_DRIVING_PARAMETERS,
    METHOD_UNITS_GET_LOCALE_CORRECTED_EXPRESSION,
    METHOD_UNITS_CONVERT,
    METHOD_UNITS_GET_TYPE_FROM_STRING,
    METHOD_UNITS_GET_STRING_FROM_TYPE,
    METHOD_UNITS_GET_STRING_FROM_VALUE,
    METHOD_UNITS_GET_PRECISE_STRING_FROM_VALUE,
];

/// A transport over immutable session state. It cannot record edits or change focus.
pub struct SessionQueryHost<'a, O, W> {
    workspace: &'a WorkspaceSession<O, W>,
}

impl<'a, O: TransactionObserver, W: WorkspaceObserver> SessionQueryHost<'a, O, W> {
    /// Borrow the session owned by the caller; example: `Client::new(SessionQueryHost::new(&workspace))`.
    pub fn new(workspace: &'a WorkspaceSession<O, W>) -> Self {
        Self { workspace }
    }

    fn active_units(&self) -> Result<&UnitsOfMeasure, Error> {
        Ok(self.active_parameters()?.units())
    }

    fn active_parameters(&self) -> Result<&ParameterCollection<u64>, Error> {
        let id = self
            .workspace
            .active_document()
            .ok_or_else(|| Error("unit query: expected an active document".into()))?;
        let session = self
            .workspace
            .session(id)
            .map_err(|error| Error(error.to_string()))?;
        Ok(&session.snapshot().parameters)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver> Caller for SessionQueryHost<'_, O, W> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_PART_DESIGN_VIEWS_GET_CAMERA => encode(&part_design_views::camera(
                self.workspace,
                decode(method, request)?,
            )?),
            METHOD_PART_DESIGN_VIEWS_LIST => encode(&part_design_views::list(self.workspace)?),
            METHOD_MODEL_STATES_LIST => encode(&model_states::list(self.workspace)?),
            METHOD_MODEL_STATES_GET_TABLE => encode(&model_states::table(self.workspace)?),
            METHOD_MODEL_STATES_VERIFY_TABLE => encode(&model_states::verify_table(
                self.workspace,
                decode(method, request)?,
                |id, candidate| {
                    self.workspace
                        .verify_unlinked_named_candidate(id, candidate)
                },
            )?),
            METHOD_MODEL_SELECTION => encode(&model_selection::read(self.workspace)?),
            METHOD_APPLICATION_API_VERSION => encode(&ApplicationApiVersionResult {
                version: oblikovati_api::VERSION.into(),
                major: i64::from(oblikovati_api::API_MAJOR),
                minor: i64::from(oblikovati_api::API_MINOR),
            }),
            METHOD_SKETCH_GET_INFERENCE_SCOPE => {
                sketch_scope::query(self.workspace, decode(method, request)?)
            }
            METHOD_SKETCH_GET_INFERENCE_OPTIONS => {
                sketch_inference::query_inference_preferences(self.workspace)
            }
            METHOD_RIBBON_LIST => encode(&ribbon::list(ribbon::key_for_document(None))),
            METHOD_CLIENT_APPS_LIST => encode(&ListClientApplicationsResult {
                clients: Some(self.workspace.connected_clients().entries().to_vec()),
            }),
            METHOD_DOCUMENTS_LIST_ATTACHMENTS => encode(&attachments::list(
                self.workspace,
                decode(method, request)?,
                None,
            )?),
            METHOD_DOCUMENTS_LIST_INTERESTS => {
                encode(&interests::list(self.workspace, decode(method, request)?)?)
            }
            METHOD_DOCUMENTS_HAS_INTEREST => {
                encode(&interests::has(self.workspace, decode(method, request)?)?)
            }
            METHOD_ATTRIBUTES_GET
            | METHOD_ATTRIBUTES_LIST
            | METHOD_ATTRIBUTES_LIST_SETS
            | METHOD_ATTRIBUTES_FIND => attributes::dispatch(self.workspace, method, request),
            METHOD_DOCUMENTS_LIST_PROPERTIES => {
                encode(&properties::list(self.workspace, decode(method, request)?)?)
            }
            METHOD_DOCUMENTS_GET_PROPERTY => {
                encode(&properties::get(self.workspace, decode(method, request)?)?)
            }
            METHOD_TRANSACTION_HISTORY => encode(&transactions::history(
                self.workspace,
                decode(method, request)?,
            )?),
            METHOD_TRANSACTION_STATE => encode(&transactions::state(self.workspace)?),
            METHOD_PARAMETERS_GET_DETAIL => encode(&detail::read(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_PARAMETERS_GET_SETTINGS => {
                encode(&collection::settings(self.active_parameters()?)?)
            }
            METHOD_PARAMETERS_GROUPS_LIST => {
                encode(&collection::groups(self.active_parameters()?)?)
            }
            METHOD_PARAMETERS_LIST => encode(&parameters::list(self.active_parameters()?)?),
            METHOD_PARAMETERS_GET => encode(&parameters::get(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_PARAMETERS_EXPORT => {
                encode(&parameter_exchange::export(self.active_parameters()?)?)
            }
            METHOD_PARAMETERS_DRIVEN_BY => encode(&parameters::neighbors(
                self.active_parameters()?,
                decode(method, request)?,
                true,
            )?),
            METHOD_PARAMETERS_DEPENDENTS => encode(&parameters::neighbors(
                self.active_parameters()?,
                decode(method, request)?,
                false,
            )?),
            METHOD_UNITS_GET_VALUE_FROM_EXPRESSION => encode(&expressions::value(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_GET_DATABASE_UNITS_FROM_EXPRESSION => encode(&expressions::database(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_IS_EXPRESSION_VALID => encode(&expressions::valid(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_COMPATIBLE_UNITS => encode(&expressions::compatible(
                self.active_parameters()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_GET_DRIVING_PARAMETERS => {
                encode(&expressions::references(decode(method, request)?)?)
            }
            METHOD_UNITS_GET_LOCALE_CORRECTED_EXPRESSION => {
                encode(&expressions::canonical(decode(method, request)?)?)
            }
            METHOD_DOCUMENTS_GET_UNITS => encode(&units::document_units(self.active_units()?)),
            METHOD_UNITS_CONVERT => encode(&units::convert(decode(method, request)?)?),
            METHOD_UNITS_GET_TYPE_FROM_STRING => {
                encode(&units::category(decode(method, request)?)?)
            }
            METHOD_UNITS_GET_STRING_FROM_TYPE => encode(&units::preferred(
                self.active_units()?,
                decode(method, request)?,
            )?),
            METHOD_UNITS_GET_STRING_FROM_VALUE => encode(&units::formatted(
                self.active_units()?,
                decode(method, request)?,
                false,
            )?),
            METHOD_UNITS_GET_PRECISE_STRING_FROM_VALUE => encode(&units::formatted(
                self.active_units()?,
                decode(method, request)?,
                true,
            )?),
            _ => Err(Error(format!(
                "method {method:?}: expected a supported read-only method from {QUERY_METHODS:?}"
            ))),
        }
    }
}

mod derived_edit;
mod drawing_annotations;
mod drawing_reference;
mod drawing_sheets;
pub use drawing_annotations::part_drawing_parts_lists;
mod drawing_styles;
mod drawing_views;
pub use drawing_views::{
    DRAWING_COMMAND_METHODS, DrawingViewProjectionAccess, prepare_base_view,
    with_current_drawing_part,
};

mod wire_json;
pub use wire_json::{decode_wire_arguments, encode_wire_reply};
pub(crate) use wire_json::{decode_wire_arguments as decode, encode_wire_reply as encode};

mod inventory;
pub use inventory::file_host_methods;

mod assets;
pub use assets::ASSET_QUERY_METHODS;

mod asset_commands;
pub use asset_commands::ASSET_COMMAND_METHODS;

mod asset_assignments;
pub use asset_assignments::ASSET_ASSIGNMENT_METHODS;
mod part_topology;
pub use feature_lifecycle::{
    FEATURE_LIFECYCLE_METHODS, apply_chamfer_distance_edit, apply_feature_distance_edit,
    apply_feature_extrusion_edit, apply_fillet_radius_edit,
};
pub use part_topology::{BODY_QUERY_METHODS, PartTopologyAccess, TopologyEntityKind};
mod part_sketch_continuity;
pub use part_sketch_continuity::PartSketchContinuity;
mod part_update;
pub use part_update::{
    CertifiedCheckpointCandidate, CertifiedCheckpointPrefix, CertifiedPartCandidate,
    CertifiedPartPrefix, PartReplayService, PartUpdateAccess, part_feature_errors,
    part_replay_is_stale, preview_part_datums, refresh_part_sketches, resolve_slot_face,
    slot_datums_for_face,
};

mod body_commands;
pub use body_commands::BODY_COMMAND_METHODS;

mod themes;
pub use themes::{THEME_QUERY_METHODS, ThemeQueryHost};

mod color_schemes;
mod keymap;
mod marking_menu;
pub use color_schemes::COLOR_SCHEME_METHODS;

#[cfg(test)]
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
mod prism_test_fixture;
