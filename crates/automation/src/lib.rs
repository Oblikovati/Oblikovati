// SPDX-License-Identifier: GPL-2.0-only

//! Typed owner-thread queries and transactional edits; native transport is a separate boundary.
pub mod conditional_prism;
mod datums;
pub mod legacy_profile;
mod part_owner;
mod reference_bytes;
pub use datums::{
    DATUM_COMMAND_METHODS, DATUM_QUERY_METHODS, requires_part_replay, work_plane_reference,
};
mod clients;
pub use clients::CLIENT_COMMAND_METHODS;
mod attachments;
mod attributes;
mod chamfer;
pub use chamfer::{PreparedChamfer, prepare_chamfer};
mod command_owner;
mod edit;
mod extrusion;
mod feature_suppression;
mod file_host;
mod fillet;
pub use extrusion::{
    ExtrudeRegionAccess, PreparedExtrusion, prepare_extrusion, validate_extrude_distance,
};
pub use fillet::{
    CertifiedFilletAccess, CertifiedFilletValidation, FilletAccess, FilletValidation,
    PreparedFillet, prepare_fillet,
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
pub use sketch_edit_state::SketchInspectionEdit;
mod sketches;
pub use edit::EDIT_METHODS;
pub use host::SessionHost;
pub use options::SKETCH_OPTION_METHODS;
pub use sketches::{SKETCH_COMMAND_METHODS, SKETCH_QUERY_METHODS};
mod collection;
mod detail;
pub use detail::parameter_detail;
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
pub const QUERY_METHODS: [&str; 36] = [
    METHOD_MODEL_STATES_LIST,
    METHOD_APPLICATION_API_VERSION,
    METHOD_CLIENT_APPS_LIST,
    METHOD_MODEL_SELECTION,
    METHOD_SKETCH_GET_INFERENCE_OPTIONS,
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
            METHOD_MODEL_STATES_LIST => encode(&model_states::list(self.workspace)?),
            METHOD_MODEL_SELECTION => encode(&model_selection::read(self.workspace)?),
            METHOD_APPLICATION_API_VERSION => encode(&ApplicationApiVersionResult {
                version: oblikovati_api::VERSION.into(),
                major: i64::from(oblikovati_api::API_MAJOR),
                minor: i64::from(oblikovati_api::API_MINOR),
            }),
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
mod drawing_reference;
mod drawing_views;
pub use drawing_views::{DRAWING_COMMAND_METHODS, DrawingViewProjectionAccess, prepare_base_view};

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
    apply_fillet_radius_edit,
};
pub use part_topology::{BODY_QUERY_METHODS, PartTopologyAccess, TopologyEntityKind};
mod part_sketch_continuity;
pub use part_sketch_continuity::PartSketchContinuity;
mod part_update;
pub use part_update::{
    CertifiedPartPrefix, PartReplayService, PartUpdateAccess, complete_part_replay,
    part_feature_errors, part_replay_is_stale, preview_part_datums, refresh_and_adopt_part,
    refresh_part_sketches, resolve_slot_face, slot_datums_for_face,
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
