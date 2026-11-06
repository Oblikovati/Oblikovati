// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::application_prompts::{
    ApplicationPrompt, PromptFrequency, PromptPreferenceAccess,
};
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Attach loaded prompt frequencies; example: `owner.with_prompt_preferences(Box::new(file))`.
    pub fn with_prompt_preferences(mut self, store: Box<dyn PromptPreferenceAccess>) -> Self {
        self.prompt_frequencies = store.prompt_preferences();
        self.prompt_store = Some(store);
        self
    }
    /// A session choice stays in memory and leaves the stored value for the next launch.
    pub(super) fn change_prompt_frequency(
        &mut self,
        prompt: ApplicationPrompt,
        frequency: PromptFrequency,
    ) -> Result<bool, String> {
        let current = self.prompt_frequencies;
        if current.frequency(prompt) == frequency {
            return Ok(false);
        }
        if let Some(store) = self.prompt_store.as_mut().filter(|_| frequency.durable()) {
            let stored = store.prompt_preferences().with(prompt, frequency);
            store.set_prompt_preferences(stored)?;
        }
        self.prompt_frequencies = current.with(prompt, frequency);
        Ok(true)
    }
}
