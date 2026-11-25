// SPDX-License-Identifier: GPL-2.0-only
#pragma once
#include "imgui_internal.h"

namespace {
constexpr float minimum_outer_dock_width = 180.0f;

ImGuiDockNode* horizontal_dock_sibling(ImGuiDockNode* node) {
    ImGuiDockNode* parent = node->ParentNode;
    if (parent->SplitAxis != ImGuiAxis_X) return nullptr;
    ImGuiDockNode* sibling = parent->ChildNodes[parent->ChildNodes[0] == node ? 1 : 0];
    return sibling && sibling->IsVisible && sibling->SizeRef.x > 0.0f ? sibling : nullptr;
}

struct OuterDockBudget {
    float available;
    float preferred;
    float minimum;
};

OuterDockBudget measure_outer_docks(ImGuiDockNode* center, float width) {
    float preferred = 0.0f, minimum = 0.0f, splitters = 0.0f;
    for (auto* node = center; node && node->ParentNode; node = node->ParentNode) {
        const auto* sibling = horizontal_dock_sibling(node);
        if (!sibling) continue;
        preferred += sibling->SizeRef.x;
        minimum += ImMin(minimum_outer_dock_width, sibling->SizeRef.x);
        splitters += 2.0f; // DOCKING_SPLITTER_SIZE in ImGui 1.89.2.
    }
    // At narrow widths, viewport space takes precedence over the dock minimums (#4063).
    const float reserved = ImMax(width * 0.55f, ImMin(520.0f, width - minimum - splitters));
    return {ImMax(0.0f, ImFloor(width - reserved) - splitters), preferred, minimum};
}

struct DockWidth {
    ImGuiID id;
    float preferred;
    float applied;
};

float dock_width_within_budget(float preferred, const OuterDockBudget& budget) {
    if (budget.preferred <= budget.available) return preferred;
    const float minimum = ImMin(minimum_outer_dock_width, preferred);
    const float remaining = budget.available - budget.minimum;
    if (remaining <= 0.0f) return minimum * budget.available / budget.minimum;
    return minimum + (preferred - minimum) * remaining / (budget.preferred - budget.minimum);
}

float limit_splitter_width(float width, float preferred, const OuterDockBudget& budget) {
    const float neighbours = budget.minimum - ImMin(minimum_outer_dock_width, preferred);
    return ImClamp(width, 1.0f, ImMax(1.0f, budget.available - neighbours));
}

void apply_outer_dock_width(ImGuiDockNode* node, float applied, ImVector<DockWidth>& widths) {
    applied = ImMax(1.0f, ImFloor(applied));
    widths.push_back({node->ID, node->SizeRef.x, applied});
    node->SizeRef.x = applied;
}

void apply_anchored_width(ImGuiDockNode* anchor, OuterDockBudget& budget, ImVector<DockWidth>& widths) {
    if (!anchor || budget.preferred <= budget.available || budget.available < budget.minimum) return;
    // Keep an explicitly dragged width before sharing the rest with neighboring docks.
    const float preferred = anchor->SizeRef.x;
    apply_outer_dock_width(anchor, limit_splitter_width(preferred, preferred, budget), widths);
    budget.available -= anchor->SizeRef.x;
    budget.preferred -= preferred;
    budget.minimum -= ImMin(minimum_outer_dock_width, preferred);
}

void apply_outer_dock_widths(ImGuiDockNode* center, ImGuiID anchored, OuterDockBudget budget,
                             ImVector<DockWidth>& widths) {
    ImGuiDockNode* anchor = nullptr;
    for (auto* node = center; anchored && node && node->ParentNode; node = node->ParentNode) {
        auto* sibling = horizontal_dock_sibling(node);
        if (sibling && sibling->ID == anchored) anchor = sibling;
    }
    apply_anchored_width(anchor, budget, widths);
    const bool anchor_applied = !widths.empty();
    for (auto* node = center; node && node->ParentNode; node = node->ParentNode) {
        auto* sibling = horizontal_dock_sibling(node);
        if (!sibling || (sibling == anchor && anchor_applied)) continue;
        apply_outer_dock_width(sibling, dock_width_within_budget(sibling->SizeRef.x, budget), widths);
    }
}

bool dragging_outer_splitter(const ImGuiDockNode* node, ImGuiWindow* host) {
    if (!host || !node->ParentNode) return false;
    // Mirrors PushID(node->ID) and GetID("##Splitter") in ImGui 1.89.2.
    return GImGui->ActiveId == ImHashStr("##Splitter", 0, host->GetID(static_cast<int>(node->ParentNode->ID)));
}

void save_splitter_changes(const ImVector<DockWidth>& widths, const OuterDockBudget& budget,
                           ImGuiWindow* host, ImGuiID& anchor) {
    for (const auto& width : widths) {
        auto* node = ImGui::DockBuilderGetNode(width.id);
        if (!node) continue;
        const float chosen = node->SizeRef.x;
        node->SizeRef.x = width.preferred;
        if (chosen == width.applied || !dragging_outer_splitter(node, host)) continue;
        // ImGui already measured the drag from the visible splitter position.
        node->SizeRef.x = limit_splitter_width(chosen, width.preferred, budget);
        anchor = width.id;
    }
}

ImGuiID draw_outer_dockspace(ImGuiID id, ImVec2 size, ImGuiID& anchor) {
    auto* center = ImGui::DockBuilderGetCentralNode(id);
    const auto budget = measure_outer_docks(center, size.x);
    ImVector<DockWidth> widths;
    if (budget.preferred > budget.available || ImGui::IsMouseDown(ImGuiMouseButton_Left)) {
        apply_outer_dock_widths(center, anchor, budget, widths);
    }
    const ImGuiID dockspace = ImGui::DockSpace(id, size,
        ImGuiDockNodeFlags_PassthruCentralNode | ImGuiDockNodeFlags_NoDockingInCentralNode);
    save_splitter_changes(widths, budget, ImGui::DockBuilderGetNode(id)->HostWindow, anchor);
    return dockspace;
}
}
