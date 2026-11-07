// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{history::SketchPlacementFieldLock, sketch::SketchPlacementField};
use oblikovati_viewport::{ViewportInput, ViewportKey, ViewportPhysicalKey};

#[derive(Default)]
pub(super) struct PlacementFieldEntry {
    active: usize,
    buffer: String,
    locked: Vec<Option<SketchPlacementFieldLock>>,
    /// Ctrl+A: the next typed character replaces the buffer.
    replace_on_type: bool,
    /// The buffer does not evaluate to a usable value, so Enter must not advance.
    invalid: bool,
    /// Line values evaluated by the owner, in working units and radians.
    values: [Option<f64>; 2],
}
/// How one numeric field reads to the GUI; example: `field.text` beside the pointer.
#[derive(Clone, Debug, PartialEq)]
pub(crate) struct PlacementFieldView {
    pub typed: Option<String>,
    pub active: bool,
    pub invalid: bool,
    pub selected: bool,
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn steering_fields_are_refused_instead_of_dropping_the_typed_value() {
        use oblikovati_model::sketch::SketchPlacementBinding::{SteeringAngle, SteeringLength};
        for binding in [SteeringAngle, SteeringLength] {
            let field = SketchPlacementField {
                label: "Seed".into(),
                value: 1.,
                witness: [[0., 0.], [1., 0.]],
                binding,
            };
            let lock = Some(SketchPlacementFieldLock {
                expression: "size".into(),
                text_point: None,
            });
            let error =
                validate_dimension_locks(std::slice::from_ref(&field), &[lock]).unwrap_err();
            assert!(error.contains("Seed") && error.contains("size") && error.contains("steering"));
            validate_dimension_locks(&[field], &[None]).unwrap();
        }
    }
    #[test]
    fn width_and_height_keep_parameter_formulas_in_field_order() {
        let mut fields = PlacementFieldEntry::default();
        for character in "size".chars() {
            assert_eq!(
                fields.input(ViewportInput::Text(character), 2),
                PlacementFieldAction::Changed
            );
        }
        assert_eq!(
            fields.key(ViewportPhysicalKey::Tab),
            PlacementFieldAction::Changed
        );
        for character in "size / 2".chars() {
            fields.input(ViewportInput::Text(character), 2);
        }
        assert_eq!(
            fields.key(ViewportPhysicalKey::Enter),
            PlacementFieldAction::Commit
        );
        let locks = fields.preview_locks();
        assert_eq!(locks[0].as_ref().unwrap().expression, "size");
        assert_eq!(locks[1].as_ref().unwrap().expression, "size / 2");
    }
    #[test]
    fn editing_and_escape_do_not_publish_partial_text_as_a_dimension() {
        let mut fields = PlacementFieldEntry::default();
        assert_eq!(
            fields.input(ViewportInput::Text('x'), 0),
            PlacementFieldAction::Ignored
        );
        fields.input(ViewportInput::Text('λ'), 1);
        fields.key(ViewportPhysicalKey::Backspace);
        assert!(!fields.engaged());
        fields.input(ViewportInput::Text('9'), 1);
        fields.key(ViewportPhysicalKey::Escape);
        assert_eq!(
            fields.key(ViewportPhysicalKey::Enter),
            PlacementFieldAction::Ignored
        );
        assert!(fields.preview_locks().iter().all(Option::is_none));
    }
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(super) enum PlacementFieldAction {
    Ignored,
    Changed,
    Commit,
}
impl PlacementFieldEntry {
    pub(super) fn input(&mut self, input: ViewportInput, count: usize) -> PlacementFieldAction {
        if count == 0 {
            return PlacementFieldAction::Ignored;
        }
        self.locked.resize(count, None);
        self.active %= count;
        match input {
            ViewportInput::Text(character) if !character.is_control() => {
                if std::mem::take(&mut self.replace_on_type) {
                    self.buffer.clear();
                }
                self.buffer.push(character);
                PlacementFieldAction::Changed
            }
            ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::A,
                pressed: true,
                modifiers,
                ..
            } if modifiers.control => self.select_all(),
            ViewportInput::PhysicalKey {
                key, pressed: true, ..
            } => self.key(key),
            ViewportInput::Key {
                key: ViewportKey::Cancel,
                pressed: true,
            } => self.key(ViewportPhysicalKey::Escape),
            _ => PlacementFieldAction::Ignored,
        }
    }
    fn select_all(&mut self) -> PlacementFieldAction {
        if self.buffer.is_empty() {
            let locked = self.locked[self.active].take();
            self.buffer = locked.map_or_else(String::new, |lock| lock.expression);
        }
        self.replace_on_type = !self.buffer.is_empty();
        PlacementFieldAction::Changed
    }
    fn key(&mut self, key: ViewportPhysicalKey) -> PlacementFieldAction {
        match key {
            ViewportPhysicalKey::Tab if self.invalid => PlacementFieldAction::Ignored,
            ViewportPhysicalKey::Tab => {
                self.lock_buffer();
                self.active = (self.active + 1) % self.locked.len();
                PlacementFieldAction::Changed
            }
            ViewportPhysicalKey::Enter | ViewportPhysicalKey::KeypadEnter if self.engaged() => {
                self.lock_buffer();
                PlacementFieldAction::Commit
            }
            _ => self.edit_buffer(key),
        }
    }
    fn edit_buffer(&mut self, key: ViewportPhysicalKey) -> PlacementFieldAction {
        if self.buffer.is_empty() {
            return PlacementFieldAction::Ignored;
        }
        match key {
            ViewportPhysicalKey::Backspace => {
                self.buffer.pop();
            }
            ViewportPhysicalKey::Escape => self.buffer.clear(),
            _ => return PlacementFieldAction::Ignored,
        }
        PlacementFieldAction::Changed
    }
    pub(super) fn prompt(&self, fields: &[SketchPlacementField]) -> String {
        fields
            .iter()
            .enumerate()
            .map(|(index, field)| self.field_prompt(index, field))
            .collect::<Vec<_>>()
            .join("  ")
    }
    fn field_prompt(&self, index: usize, field: &SketchPlacementField) -> String {
        let value = self
            .field_expression(index)
            .map(str::to_owned)
            .unwrap_or_else(|| field.value.to_string());
        let text = format!("{}: {value}", field.label);
        if index == self.active {
            format!("[{text}]")
        } else {
            text
        }
    }
    fn field_expression(&self, index: usize) -> Option<&str> {
        if index == self.active && !self.buffer.is_empty() {
            return Some(&self.buffer);
        }
        self.locked
            .get(index)
            .and_then(Option::as_ref)
            .map(|lock| lock.expression.as_str())
    }
    fn lock_buffer(&mut self) {
        // An invalid value stays editable; only a value that evaluates may hold the field.
        if self.buffer.is_empty() || self.invalid {
            return;
        }
        self.replace_on_type = false;
        self.locked[self.active] = Some(SketchPlacementFieldLock {
            expression: std::mem::take(&mut self.buffer),
            text_point: None,
        });
    }
    /// An arc's next stage takes the next value: what was typed so far is kept as it stands.
    pub(super) fn advance_to_stage(&mut self, stage: usize) {
        self.locked.resize(stage.max(self.locked.len()), None);
        if !self.locked.is_empty() {
            self.lock_buffer();
        }
        self.active = stage.saturating_sub(1);
    }
    pub(super) fn engaged(&self) -> bool {
        !self.buffer.is_empty() || self.locked.iter().any(Option::is_some)
    }
    /// Evaluate what is typed; the owner holds the parameters. Example: on each field change.
    pub(super) fn evaluate_line_fields(&mut self, evaluate: &dyn Fn(usize, &str) -> Option<f64>) {
        self.invalid = false;
        for index in 0..2 {
            let source = self.field_expression(index).map(str::to_owned);
            let value = source.as_deref().and_then(|text| evaluate(index, text));
            self.values[index] = value;
            self.invalid |= index == self.active && source.is_some() && value.is_none();
        }
    }
    pub(super) fn line_values(&self) -> [Option<f64>; 2] {
        self.values
    }
    /// Field order is length then angle; `keep` false moves the geometry but retains no dimension.
    pub(super) fn line_locks(&self, keep: bool) -> [Option<SketchPlacementFieldLock>; 2] {
        std::array::from_fn(|index| {
            // An invalid value must not reach the preview: a zero length would collapse the line.
            self.values[index]?;
            let expression = self.field_expression(index).filter(|_| keep)?;
            Some(SketchPlacementFieldLock {
                expression: expression.to_owned(),
                text_point: None,
            })
        })
    }
    pub(super) fn has_invalid_value(&self) -> bool {
        self.invalid
    }
    pub(super) fn view(&self, index: usize) -> PlacementFieldView {
        PlacementFieldView {
            typed: self.field_expression(index).map(str::to_owned),
            active: index == self.active,
            invalid: index == self.active && self.invalid,
            selected: index == self.active && self.replace_on_type,
        }
    }
    pub(super) fn preview_locks(&self) -> Vec<Option<SketchPlacementFieldLock>> {
        let mut locks = self.locked.clone();
        if !self.buffer.is_empty() {
            locks[self.active] = Some(SketchPlacementFieldLock {
                expression: self.buffer.clone(),
                text_point: None,
            });
        }
        locks
    }
}
pub(super) fn validate_dimension_locks(
    fields: &[SketchPlacementField],
    locks: &[Option<SketchPlacementFieldLock>],
) -> Result<(), String> {
    use oblikovati_model::sketch::SketchPlacementBinding::{SteeringAngle, SteeringLength};
    for (field, lock) in fields.iter().zip(locks) {
        if let Some(lock) = lock
            && matches!(field.binding, SteeringAngle | SteeringLength)
        {
            return Err(format!(
                "placement field {:?}, expression {:?}: expected a driving dimension; steering input requires seed editing",
                field.label, lock.expression
            ));
        }
    }
    Ok(())
}
