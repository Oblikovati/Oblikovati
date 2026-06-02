// SPDX-License-Identifier: GPL-2.0-only

// Package app is the application-shell logic: the command framework, interactive
// tools, selection and interaction pipeline, and the session that ties them to the
// open documents. It is the modern realization of Inventor's CommandManager /
// ControlDefinitions / SelectSet / InteractionEvents (plan M05), and the layer Dear
// ImGui renders from each frame (ADR-0004).
//
// Crucially it is PURE GO with no ImGui/Vulkan dependency: per ADR-0014 all UI logic
// lives "below the GPU line" as functions over data, and per ADR-0004 the state
// lives in the model, not the widgets. So the whole interactive surface is driven by
// synthetic input in tests — a [Session] exposes Click/PressKey/Invoke so a test can
// "operate the UI" (pick a profile, type a distance, hit OK) and assert the
// resulting geometry, with no GPU. The Vulkan viewport and ImGui chrome are a thin
// rendering layer added later behind a backend interface.
//
// Inventor fidelity: command aliases, mouse behavior (LMB select, RMB marking menu,
// MMB-drag orbit), and the OK/Apply/Cancel tool flow follow Autodesk Inventor 2026.
package app
