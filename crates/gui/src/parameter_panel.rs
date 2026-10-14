// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    GuiParameterAck, GuiParameterDrafts, GuiParameterRow, GuiParameterSnapshot,
    GuiParameterTextEdit, GuiParameterTextField, GuiParameterTextTarget,
};
use imgui::{Condition, TableFlags, Ui};
mod control_identity;
mod create;
mod group;
mod group_manager;
mod linked_tables;
use control_identity::parameter_controls_identity;
pub use group::GuiParameterGroupEdit;
mod request_state;
mod rows;
mod tolerance;
mod value_list;
pub use create::{GuiParameterCreation, GuiParameterInitialValue};
pub use tolerance::GuiParameterToleranceEdit;
pub use value_list::GuiParameterChoiceEdit;
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiParameterLinkEdit {
    pub document: u64,
    pub source_document: u64,
    pub request: u64,
    pub linked: Vec<u64>,
}
impl GuiParameterLinkEdit {
    /// Check the owned source selection before a link command; example: `edit.validate()?`.
    pub fn validate(&self) -> Result<(), String> {
        if self.document == 0
            || self.source_document == 0
            || self.document == self.source_document
            || self.request == 0
        {
            return Err(format!(
                "GUI link edit {self:?}: expected distinct nonzero document identities and a positive request"
            ));
        }
        let unique = self
            .linked
            .iter()
            .collect::<std::collections::BTreeSet<_>>();
        if unique.len() != self.linked.len() {
            return Err(format!(
                "GUI link edit {:?}: expected unique source parameter identities",
                self.linked
            ));
        }
        Ok(())
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiParameterToggleField {
    Boolean,
    Key,
    Export,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiParameterRowOperation {
    CopyToUser,
    Delete,
    EditValueList,
    EditTolerance,
    AddToGroup,
    RemoveFromGroups,
    ClearValueList,
}
#[derive(Clone, Debug, PartialEq)]
pub enum GuiParameterAction {
    GroupManagement(GuiParameterGroupAction),
    Link(GuiParameterLinkEdit),
    DeleteLinkedTable {
        document: u64,
        table: i64,
    },
    Add(GuiParameterCreation),
    Row {
        document: u64,
        parameter: u64,
        operation: GuiParameterRowOperation,
    },
    Text(GuiParameterTextEdit),
    Choices(GuiParameterChoiceEdit),
    Tolerance(GuiParameterToleranceEdit),
    Group(GuiParameterGroupEdit),
    Toggle {
        document: u64,
        parameter: u64,
        field: GuiParameterToggleField,
        value: bool,
    },
}
/// A document-level change to immutable-keyed parameter groups.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiParameterGroupAction {
    Create {
        document: u64,
        internal_name: String,
        display_name: String,
        client_id: String,
    },
    Rename {
        document: u64,
        internal_name: String,
        display_name: String,
    },
    Delete {
        document: u64,
        internal_name: String,
        delete_parameters: bool,
    },
    RemoveMember {
        document: u64,
        internal_name: String,
        parameter: u64,
    },
}
impl GuiParameterGroupAction {
    /// Reject malformed names before they reach the document owner; example: `action.validate()?`.
    pub fn validate(&self) -> Result<(), String> {
        let (document, key, display, client) = match self {
            Self::Create {
                document,
                internal_name,
                display_name,
                client_id,
            } => (
                *document,
                internal_name,
                Some(display_name),
                Some(client_id),
            ),
            Self::Rename {
                document,
                internal_name,
                display_name,
            } => (*document, internal_name, Some(display_name), None),
            Self::Delete {
                document,
                internal_name,
                ..
            } => (*document, internal_name, None, None),
            Self::RemoveMember {
                document,
                internal_name,
                parameter,
            } => {
                if *parameter == 0 {
                    return Err(format!(
                        "GUI parameter group action {self:?}: expected a nonzero member identity"
                    ));
                }
                (*document, internal_name, None, None)
            }
        };
        if document == 0
            || key.is_empty()
            || key.contains('\0')
            || display.is_some_and(|value| value.is_empty() || value.contains('\0'))
            || client.is_some_and(|value| value.contains('\0'))
        {
            return Err(format!(
                "GUI parameter group action {self:?}: expected a nonzero document, nonempty NUL-free key and valid presentation text"
            ));
        }
        Ok(())
    }
    pub fn document(&self) -> u64 {
        match self {
            Self::Create { document, .. }
            | Self::Rename { document, .. }
            | Self::Delete { document, .. }
            | Self::RemoveMember { document, .. } => *document,
        }
    }
}
pub(crate) struct ParameterPanel {
    open: bool,
    snapshot: Option<GuiParameterSnapshot>,
    search: String,
    drafts: GuiParameterDrafts,
    issue: Option<String>,
    metadata_issue: Option<String>,
    creation: create::ParameterCreationForm,
    value_list: value_list::ParameterValueListEditor,
    tolerance: tolerance::ParameterToleranceEditor,
    group: group::ParameterGroupEditor,
    group_manager: group_manager::ParameterGroupManager,
    link_picker: linked_tables::LinkPicker,
}
impl ParameterPanel {
    pub(crate) fn new() -> Result<Self, String> {
        Ok(Self {
            open: false,
            snapshot: None,
            search: String::new(),
            drafts: GuiParameterDrafts::new(1024 * 1024, 4096)?,
            issue: None,
            metadata_issue: None,
            creation: Default::default(),
            value_list: Default::default(),
            tolerance: Default::default(),
            group: Default::default(),
            group_manager: Default::default(),
            link_picker: Default::default(),
        })
    }
    pub(crate) fn replace_snapshot(
        &mut self,
        snapshot: Option<GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        self.drafts.synchronize(snapshot.as_ref())?;
        self.creation
            .synchronize(snapshot.as_ref().map(|view| view.document));
        self.value_list.synchronize(snapshot.as_ref());
        self.tolerance.synchronize(snapshot.as_ref());
        self.group.synchronize(snapshot.as_ref());
        self.group_manager.synchronize(snapshot.as_ref());
        self.link_picker.synchronize(snapshot.as_ref());
        let changed = self.snapshot != snapshot || self.metadata_issue.is_some();
        self.metadata_issue = None;
        self.snapshot = snapshot;
        Ok(changed)
    }
    pub(crate) fn report_metadata_issue(&mut self, issue: String) -> Result<bool, String> {
        let changed = self.snapshot.is_some() || self.metadata_issue.as_ref() != Some(&issue);
        self.replace_snapshot(None)?;
        self.metadata_issue = Some(issue);
        Ok(changed)
    }
    pub(crate) fn set_open(&mut self, open: bool) {
        self.open = open;
    }
    pub(crate) fn is_open(&self) -> bool {
        self.open
    }
    pub(crate) fn acknowledge(
        &mut self,
        edit: &GuiParameterTextEdit,
        result: Result<(), String>,
    ) -> GuiParameterAck {
        self.drafts.acknowledge(edit, result)
    }
    pub(crate) fn acknowledge_action(
        &mut self,
        action: &GuiParameterAction,
        result: Result<(), String>,
    ) {
        match action {
            GuiParameterAction::Group(edit) => {
                self.group.acknowledge(edit, result);
            }
            GuiParameterAction::Tolerance(edit) => {
                self.tolerance.acknowledge(edit, result);
            }
            GuiParameterAction::Choices(edit) => {
                self.value_list.acknowledge(edit, result);
            }
            GuiParameterAction::Add(creation) => {
                self.creation.acknowledge(creation, result);
            }
            GuiParameterAction::Link(edit) => self.link_picker.acknowledge(edit, result),
            GuiParameterAction::Text(edit) => {
                self.acknowledge(edit, result);
            }
            GuiParameterAction::GroupManagement(edit) => {
                if let Some(issue) = self.group_manager.acknowledge(edit, result) {
                    self.issue = Some(issue);
                }
            }
            GuiParameterAction::Toggle { .. }
            | GuiParameterAction::Row { .. }
            | GuiParameterAction::DeleteLinkedTable { .. } => self.issue = result.err(),
        }
    }
    pub(crate) fn draw(&mut self, ui: &Ui) -> Vec<GuiParameterAction> {
        let mut actions = Vec::new();
        if !self.open {
            return actions;
        }
        let mut open = self.open;
        ui.window("Parameters")
            .size([920., 480.], Condition::FirstUseEver)
            .opened(&mut open)
            .build(|| self.draw_body(ui, &mut actions));
        if !open {
            self.open = false;
        }
        self.consume_editor_actions(&mut actions);
        self.value_list.draw(ui, &mut actions);
        self.tolerance.draw(ui, &mut actions);
        self.group.draw(ui, &mut actions);
        self.link_picker
            .draw(ui, self.snapshot.as_ref(), &mut actions);
        actions
    }
    fn consume_editor_actions(&mut self, actions: &mut Vec<GuiParameterAction>) {
        actions.retain(|action| {
            let GuiParameterAction::Row {
                document,
                parameter,
                operation,
            } = action
            else {
                return true;
            };
            if !matches!(
                operation,
                GuiParameterRowOperation::EditValueList
                    | GuiParameterRowOperation::EditTolerance
                    | GuiParameterRowOperation::AddToGroup
            ) {
                return true;
            }
            self.issue = self
                .open_row_editor(*document, *parameter, *operation)
                .err();
            false
        });
    }
    fn open_row_editor(
        &mut self,
        document: u64,
        parameter: u64,
        operation: GuiParameterRowOperation,
    ) -> Result<(), String> {
        let row = self
            .snapshot
            .as_ref()
            .filter(|view| view.document == document)
            .and_then(|view| view.rows.iter().find(|row| row.parameter == parameter))
            .ok_or_else(|| {
                format!(
                    "GUI row editor target {document}/{parameter}: expected a current parameter row"
                )
            })?;
        match operation {
            GuiParameterRowOperation::AddToGroup => {
                self.group.open(document, row);
                Ok(())
            }
            GuiParameterRowOperation::EditValueList => self.value_list.open(document, row),
            GuiParameterRowOperation::EditTolerance => self.tolerance.open(document, row),
            found => Err(format!(
                "GUI row editor operation {found:?}: expected a value-list, tolerance, or group editor"
            )),
        }
    }
    fn draw_body(&mut self, ui: &Ui, actions: &mut Vec<GuiParameterAction>) {
        ui.set_next_item_width(-1.);
        ui.input_text("##parameter-search", &mut self.search)
            .build();
        if let Some(issue) = &self.metadata_issue {
            ui.text("Parameter view unavailable:");
            ui.text_wrapped(issue.replace('\0', "\\0"));
            return;
        }
        let Some(snapshot) = &self.snapshot else {
            ui.text("Open a part or assembly to edit parameters.");
            return;
        };
        let filter = self.search.trim().to_lowercase();
        draw_parameter_section(
            ui,
            "Model Parameters",
            false,
            snapshot,
            &filter,
            &mut self.drafts,
            actions,
        );
        draw_parameter_section(
            ui,
            "User Parameters",
            true,
            snapshot,
            &filter,
            &mut self.drafts,
            actions,
        );
        self.creation.draw(ui, actions);
        linked_tables::draw_linked_tables(ui, snapshot, actions, &mut self.link_picker);
        self.group_manager.draw(ui, snapshot, actions);
        if let Some(issue) = &self.issue {
            ui.text_wrapped(issue.replace('\0', "\\0"));
        }
        if ui.button("Done") {
            self.open = false;
        }
    }
}
fn draw_parameter_section(
    ui: &Ui,
    title: &str,
    user: bool,
    snapshot: &GuiParameterSnapshot,
    filter: &str,
    drafts: &mut GuiParameterDrafts,
    actions: &mut Vec<GuiParameterAction>,
) {
    ui.separator();
    ui.text(title);
    let rows: Vec<_> = snapshot
        .rows
        .iter()
        .filter(|row| parameter_matches(row, user, filter))
        .collect();
    if rows.is_empty() {
        ui.text("(none)");
        return;
    }
    let height = ui.frame_height_with_spacing() * (rows.len().min(8) + 1) as f32;
    let flags =
        TableFlags::BORDERS | TableFlags::ROW_BG | TableFlags::SCROLL_Y | TableFlags::RESIZABLE;
    let Some(_table) = ui.begin_table_with_sizing(title, 8, flags, [0., height], 0.) else {
        return;
    };
    configure_parameter_columns(ui);
    for row in rows {
        rows::draw_parameter_row(ui, snapshot.document, row, drafts, actions);
    }
}
fn configure_parameter_columns(ui: &Ui) {
    for column in [
        "Name", "Unit", "Equation", "Value", "Tol.", "Key", "Export", "Comment",
    ] {
        ui.table_setup_column(column);
    }
    ui.table_setup_scroll_freeze(0, 1);
    ui.table_headers_row();
}
fn parameter_matches(row: &GuiParameterRow, user: bool, filter: &str) -> bool {
    let information = &row.detail.parameter_info;
    if (information.kind == "user") != user {
        return false;
    }
    [
        &information.name,
        &information.expression,
        &row.detail.comment,
    ]
    .iter()
    .any(|text| text.to_lowercase().contains(filter))
}
#[cfg(test)]
#[path = "../tests/parameters/panel.rs"]
mod tests;
