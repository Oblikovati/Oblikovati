// SPDX-License-Identifier: GPL-2.0-only
//! How often the application asks before it closes work; document edits and undo do not change it.
use serde::{Deserialize, Serialize};

/// The prompts a user can quiet from their own frequency list.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ApplicationPrompt {
    SaveChanges,
    ExitEnvironment,
}
/// One prompt's frequency; `SuppressForSession` never reaches a stored preference.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PromptFrequency {
    #[default]
    AlwaysShow,
    OncePerOperation,
    SuppressForSession,
    SuppressPermanently,
}
impl PromptFrequency {
    /// Every choice in list order; example: `PromptFrequency::ALL[0]` is `AlwaysShow`.
    pub const ALL: [Self; 4] = [
        Self::AlwaysShow,
        Self::OncePerOperation,
        Self::SuppressForSession,
        Self::SuppressPermanently,
    ];
    /// Whether the prompt stays closed; example: `frequency.suppressed()`.
    pub fn suppressed(self) -> bool {
        matches!(self, Self::SuppressForSession | Self::SuppressPermanently)
    }
    /// Whether the choice outlives this session; example: `frequency.durable()`.
    pub fn durable(self) -> bool {
        self != Self::SuppressForSession
    }
}
/// The frequency of each prompt.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default)]
pub struct PromptPreferences {
    pub save_changes: PromptFrequency,
    pub exit_environment: PromptFrequency,
}
impl PromptPreferences {
    /// Read one prompt; example: `prompts.frequency(ApplicationPrompt::SaveChanges)`.
    pub fn frequency(self, prompt: ApplicationPrompt) -> PromptFrequency {
        match prompt {
            ApplicationPrompt::SaveChanges => self.save_changes,
            ApplicationPrompt::ExitEnvironment => self.exit_environment,
        }
    }
    /// Copy with one prompt changed; example: `prompts.with(prompt, PromptFrequency::AlwaysShow)`.
    pub fn with(mut self, prompt: ApplicationPrompt, frequency: PromptFrequency) -> Self {
        match prompt {
            ApplicationPrompt::SaveChanges => self.save_changes = frequency,
            ApplicationPrompt::ExitEnvironment => self.exit_environment = frequency,
        }
        self
    }
    /// Refuse a value a file must not hold; example: `prompts.validate_stored()?`.
    pub fn validate_stored(self) -> Result<(), String> {
        for frequency in [self.save_changes, self.exit_environment] {
            if !frequency.durable() {
                return Err(format!(
                    "stored prompt frequency {frequency:?}: expected a choice that outlives the session"
                ));
            }
        }
        Ok(())
    }
}
/// An injected preference owner publishes values only after successful storage.
pub trait PromptPreferenceAccess {
    /// Read stored frequencies; example: `store.prompt_preferences().save_changes`.
    fn prompt_preferences(&self) -> PromptPreferences;
    /// Persist stored frequencies; example: `store.set_prompt_preferences(prompts)?`.
    fn set_prompt_preferences(&mut self, prompts: PromptPreferences) -> Result<(), String>;
}

#[cfg(test)]
#[path = "../tests/application_prompts/model.rs"]
mod tests;
