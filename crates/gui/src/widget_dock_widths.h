// SPDX-License-Identifier: GPL-2.0-only
#pragma once
#include "imgui_internal.h"

namespace {
constexpr float minimum_outer_dock_width = 180.0f;
// DockNodeTreeUpdateSplitter and SplitterBehavior use these values together.
static_assert(IMGUI_VERSION_NUM == 18920, "Review outer splitter geometry for this ImGui version");
constexpr float outer_splitter_width = 2.0f;
constexpr float outer_splitter_hover_padding = 4.0f;

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
        splitters += outer_splitter_width;
    }
    // At narrow widths, viewport space takes precedence over the dock minimums (#4063).
    const float reserved = ImMax(width * 0.55f, ImMin(520.0f, width - minimum - splitters));
    return {ImMax(0.0f, ImFloor(width - reserved) - splitters), preferred, minimum};
}

struct DockWidth {
    ImGuiID id;
    float preferred;
};

float dock_width_within_budget(float preferred, const OuterDockBudget& budget) {
    if (budget.preferred <= budget.available) return preferred;
    const float minimum = ImMin(minimum_outer_dock_width, preferred);
    const float remaining = budget.available - budget.minimum;
    if (remaining <= 0.0f) return minimum * budget.available / budget.minimum;
    return minimum + (preferred - minimum) * remaining / (budget.preferred - budget.minimum);
}

float outer_dock_width_limit(float preferred, const OuterDockBudget& budget) {
    const float neighbours = budget.minimum - ImMin(minimum_outer_dock_width, preferred);
    return ImMax(1.0f, budget.available - neighbours);
}

void apply_outer_dock_width(ImGuiDockNode* node, float applied, ImVector<DockWidth>& widths) {
    applied = ImMax(1.0f, ImFloor(applied));
    widths.push_back({node->ID, node->SizeRef.x});
    node->SizeRef.x = applied;
}

void apply_anchored_width(ImGuiDockNode* anchor, OuterDockBudget& budget, ImVector<DockWidth>& widths) {
    if (!anchor || budget.preferred <= budget.available || budget.available < budget.minimum) return;
    // Keep an explicitly dragged width before sharing the rest with neighboring docks.
    const float preferred = anchor->SizeRef.x;
    apply_outer_dock_width(anchor, ImMin(preferred, outer_dock_width_limit(preferred, budget)), widths);
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

ImGuiDockNode* dragged_outer_dock(ImGuiDockNode* center, ImGuiWindow* host) {
    if (!host || !ImGui::IsMouseDown(ImGuiMouseButton_Left)) return nullptr;
    for (auto* node = center; node && node->ParentNode; node = node->ParentNode) {
        auto* sibling = horizontal_dock_sibling(node);
        if (!sibling) continue;
        // Mirrors PushID(node->ID) and GetID("##Splitter") in ImGui.
        const auto seed = host->GetID(static_cast<int>(node->ParentNode->ID));
        if (GImGui->ActiveId == ImHashStr("##Splitter", 0, seed)) return sibling;
    }
    return nullptr;
}

void restore_outer_dock_widths(const ImVector<DockWidth>& widths) {
    for (const auto& width : widths) {
        if (auto* node = ImGui::DockBuilderGetNode(width.id)) node->SizeRef.x = width.preferred;
    }
}

bool outer_dock_is_left(const ImGuiDockNode* node) {
    return node->ParentNode->ChildNodes[0] == node;
}

float outer_dock_edge(const ImGuiDockNode* dock, float width, bool left) {
    float edge = ImGui::GetCursorScreenPos().x + (left ? 0.0f : ImFloor(width));
    for (auto* branch = dock->ParentNode; branch->ParentNode; branch = branch->ParentNode) {
        const auto* sibling = horizontal_dock_sibling(branch);
        if (sibling && outer_dock_is_left(sibling) == left) {
            const float space = sibling->SizeRef.x + outer_splitter_width;
            edge += left ? space : -space;
        }
    }
    return edge;
}

void apply_dragged_width(ImGuiDockNode* dock, ImGuiDockNode* center, float width,
                         ImGuiID& anchor, ImVector<DockWidth>& widths, const OuterDockBudget& budget) {
    const bool left = outer_dock_is_left(dock);
    const float edge = outer_dock_edge(dock, width, left);
    const float pointer = GImGui->IO.MousePos.x - GImGui->ActiveIdClickOffset.x + outer_splitter_hover_padding;
    const float requested = left ? pointer - edge : edge - pointer - outer_splitter_width;
    const float current = dock->SizeRef.x;
    const float minimum = ImMin(current, ImGui::GetStyle().WindowMinSize.x);
    const float maximum = budget.available < budget.minimum ? current : outer_dock_width_limit(current, budget);
    const float chosen = ImFloor(ImClamp(requested, minimum, ImMax(minimum, maximum)));
    if (chosen == current) return;
    restore_outer_dock_widths(widths);
    widths.resize(0);
    dock->SizeRef.x = chosen;
    anchor = dock->ID;
    apply_outer_dock_widths(center, anchor, measure_outer_docks(center, width), widths);
}

void align_native_splitter(const ImGuiDockNode* dock, float width) {
    const bool left = outer_dock_is_left(dock);
    const float edge = outer_dock_edge(dock, width, left);
    const float boundary = left ? edge + dock->SizeRef.x : edge - dock->SizeRef.x - outer_splitter_width;
    // Leave ImGui in charge of the grab and cursor. Its resize target must match the
    // already budgeted frame, including when the real pointer is past the limit.
    GImGui->ActiveIdClickOffset.x = GImGui->IO.MousePos.x - boundary + outer_splitter_hover_padding;
}

ImGuiID draw_outer_dockspace(ImGuiID id, ImVec2 size, ImGuiID& anchor) {
    auto* center = ImGui::DockBuilderGetCentralNode(id);
    const auto budget = measure_outer_docks(center, size.x);
    auto* dock = dragged_outer_dock(center, ImGui::DockBuilderGetNode(id)->HostWindow);
    const ImGuiID active = GImGui->ActiveId;
    const float click_offset = GImGui->ActiveIdClickOffset.x;
    ImVector<DockWidth> widths;
    if (budget.preferred > budget.available || dock) {
        apply_outer_dock_widths(center, anchor, budget, widths);
    }
    if (dock) {
        apply_dragged_width(dock, center, size.x, anchor, widths, budget);
        align_native_splitter(dock, size.x);
    }
    const ImGuiID dockspace = ImGui::DockSpace(id, size,
        ImGuiDockNodeFlags_PassthruCentralNode | ImGuiDockNodeFlags_NoDockingInCentralNode);
    if (dock && GImGui->ActiveId == active) GImGui->ActiveIdClickOffset.x = click_offset;
    restore_outer_dock_widths(widths);
    return dockspace;
}
}
