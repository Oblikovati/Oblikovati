// SPDX-License-Identifier: GPL-2.0-only
#include "imgui.h"
#include "imgui_internal.h"
#include <cstddef>
#include <cstdint>

// #3803: Rust C aggregates and MSVC ARM64 C++ vector classes use different registers.
// Only scalars and byte pointers cross this boundary; C++ constructs its own vectors.
namespace {
ImGuiID workspace_dockspace_id() {
    // Keep the original host ID so saved inner layouts survive reparenting (#4071).
    return ImHashStr("##workspace-dockspace", 0, ImHashStr("##workspace-docking-host"));
}
// Mirrors DockNodeCalcDropRectsAndTestMousePos in vendored Dear ImGui 1.89.2-docking (#4071).
static_assert(IMGUI_VERSION_NUM == 18920, "Review outer dock target geometry for the new ImGui version (#4071)");
bool is_outer_dock_target(const ImGuiWindow* outer, const ImVec2 mouse) {
    const ImVec2 size = outer->Size;
    const float half = ImMin(ImGui::GetFontSize() * 1.5f,
                             ImMax(ImGui::GetFontSize() * 0.5f, ImMin(size.x, size.y) / 8.0f));
    const float wide = ImFloor(half * 1.5f);
    const float narrow = ImFloor(half * 0.8f);
    const ImVec2 center(ImFloor(outer->Pos.x + size.x * 0.5f), ImFloor(outer->Pos.y + size.y * 0.5f));
    const bool top_or_bottom = ImFabs(mouse.x - center.x) <= wide &&
        (ImFabs(mouse.y - outer->Pos.y - narrow) <= narrow ||
         ImFabs(mouse.y - outer->Pos.y - size.y + narrow) <= narrow);
    const bool left_or_right = ImFabs(mouse.y - center.y) <= wide &&
        (ImFabs(mouse.x - outer->Pos.x - narrow) <= narrow ||
         ImFabs(mouse.x - outer->Pos.x - size.x + narrow) <= narrow);
    return top_or_bottom || left_or_right;
}

ImGuiDockNode* horizontal_dock_sibling(ImGuiDockNode* node) {
    ImGuiDockNode* parent = node->ParentNode;
    if (parent->SplitAxis != ImGuiAxis_X) return nullptr;
    ImGuiDockNode* sibling = parent->ChildNodes[parent->ChildNodes[0] == node ? 1 : 0];
    return sibling && sibling->IsVisible ? sibling : nullptr;
}

float outer_dock_width_scale(ImGuiDockNode* center, float width) {
    float preferred = 0.0f;
    float splitters = 0.0f;
    for (auto* node = center; node && node->ParentNode; node = node->ParentNode) {
        if (const auto* sibling = horizontal_dock_sibling(node)) {
            preferred += sibling->SizeRef.x;
            splitters += 2.0f; // DOCKING_SPLITTER_SIZE in ImGui 1.89.2.
        }
    }
    // Leave 256 logical pixels for the Part at the minimum desktop width.
    // Smaller dock hosts may still reserve two fifths of their width (#4063).
    const float reserved = ImMin(256.0f, width * 0.4f);
    const float budget = ImMax(0.0f, ImFloor(width - reserved) - splitters);
    return preferred > budget ? budget / preferred : 1.0f;
}

struct CompactDockWidth {
    ImGuiID node;
    float preferred;
    float applied;
};

void compact_outer_docks(ImGuiDockNode* center, float width, ImVector<CompactDockWidth>& widths) {
    const float scale = outer_dock_width_scale(center, width);
    if (scale >= 1.0f) return;
    for (auto* node = center; node && node->ParentNode; node = node->ParentNode) {
        auto* sibling = horizontal_dock_sibling(node);
        if (!sibling || sibling->SizeRef.x <= 0.0f) continue;
        const float applied = ImMax(1.0f, ImFloor(sibling->SizeRef.x * scale));
        widths.push_back({sibling->ID, sibling->SizeRef.x, applied});
        sibling->SizeRef.x = applied;
    }
}

void restore_outer_dock_widths(const ImVector<CompactDockWidth>& widths) {
    for (const auto& width : widths) {
        auto* node = ImGui::DockBuilderGetNode(width.node);
        if (!node) continue;
        // Keep an intentional splitter change while discarding the temporary cap.
        const float delta = node->SizeRef.x - width.applied;
        node->SizeRef.x = ImMax(1.0f, width.preferred + delta);
    }
}

void draw_ribbon_icon(const char* label, std::uint32_t texture, ImVec2 icon, bool large, bool show_caption) {
    const ImVec2 lo = ImGui::GetItemRectMin(), hi = ImGui::GetItemRectMax();
    const char* end = ImGui::FindRenderedTextEnd(label);
    const ImVec2 text = ImGui::CalcTextSize(label, end);
    const ImVec2 at(!show_caption || large ? (lo.x + hi.x - icon.x) * 0.5f : lo.x + 2.0f,
                    !show_caption ? (lo.y + hi.y - icon.y) * 0.5f :
                    large ? lo.y + 2.0f : (lo.y + hi.y - icon.y) * 0.5f);
    const ImVec2 caption(large ? (lo.x + hi.x - text.x) * 0.5f : at.x + icon.x + 3.0f,
                         large ? hi.y - text.y - 2.0f : (lo.y + hi.y - text.y) * 0.5f);
    ImDrawList* draw = ImGui::GetWindowDrawList();
    draw->PushClipRect(lo, hi, true);
    draw->AddImage(reinterpret_cast<ImTextureID>(static_cast<std::uintptr_t>(texture)), at,
                   ImVec2(at.x + icon.x, at.y + icon.y), ImVec2(0,0), ImVec2(1,1), ImGui::GetColorU32(ImVec4(1,1,1,1)));
    if (show_caption) draw->AddText(caption, ImGui::GetColorU32(ImGuiCol_Text), label, end);
    draw->PopClipRect();
}
std::uint32_t dock_panel_menu() {
    if (!ImGui::BeginPopup("##dock-panel-menu-popup")) return 0;
    const char* labels[] = {"Model Browser", "Log Panel", "Favorites", "Material Browser",
                            "Appearance Browser", "Parameters", "Selection Filter", "History Browser"};
    std::uint32_t actions = 0;
    for (std::uint32_t index = 0; index < sizeof(labels) / sizeof(labels[0]); ++index) {
        if (ImGui::MenuItem(labels[index])) actions |= 1u << index;
    }
    ImGui::EndPopup();
    return actions;
}
}
extern "C" {
float oblikovati_gui_push_text_line_advance(float factor) {
    ImFont* font = ImGui::GetFont();
    const float original = font->FontSize;
    // ImGui uses the effective font size as the row height. Changing the nominal
    // and effective sizes together keeps the baked glyph advances and quads at the same scale.
    font->FontSize = original * factor;
    ImGui::PushFont(font);
    return original;
}

void oblikovati_gui_pop_text_line_advance(float original) {
    ImGui::GetFont()->FontSize = original;
    ImGui::PopFont();
}

// InputTextMultiline draws into a child window. Keep the tint in that draw list
// so the widget's scroll clipping and the dialog's popup order still apply.
void oblikovati_gui_shade_linked_field(const char* label, const char* line,
                                      std::size_t prefix, std::size_t end, std::size_t row) {
    ImGuiContext& context = *ImGui::GetCurrentContext();
    ImGuiWindow* parent = context.CurrentWindow;
    if (parent->DC.ChildWindows.empty()) return;
    ImGuiWindow* child = parent->DC.ChildWindows.back();
    const ImGuiID input = parent->GetID(label);
    if (child->ChildId != input || child->SkipItems) return;
    const ImFont* font = context.Font;
    const float size = context.FontSize;
    const ImGuiInputTextState* state = ImGui::GetInputTextState(input);
    const float scroll_x = state && context.ActiveId == input ? state->ScrollX : 0.0f;
    const ImVec2 origin(child->InnerRect.Min.x + context.Style.FramePadding.x - scroll_x,
                        child->InnerRect.Min.y + context.Style.FramePadding.y - child->Scroll.y);
    const float left = origin.x + font->CalcTextSizeA(size, FLT_MAX, 0.0f, line, line + prefix).x;
    const float right = origin.x + font->CalcTextSizeA(size, FLT_MAX, 0.0f, line, line + end).x;
    const float top = origin.y + size * static_cast<float>(row);
    child->DrawList->PushClipRect(child->InnerClipRect.Min, child->InnerClipRect.Max, true);
    child->DrawList->AddRectFilled(ImVec2(left, top), ImVec2(right, top + size),
                                  ImGui::GetColorU32(ImGuiCol_Text, 0.22f));
    child->DrawList->PopClipRect();
}

bool oblikovati_gui_window_is_moving() {
    return ImGui::GetCurrentContext()->MovingWindow != nullptr;
}

std::uint32_t oblikovati_gui_workspace_dockspace(float width, float height) {
    const ImGuiID id = workspace_dockspace_id();
    ImGuiContext* context = ImGui::GetCurrentContext();
    ImGuiWindow* outer = ImGui::FindWindowByName("##workspace-outer-host");
    const bool route_outer = context->MovingWindow && outer && is_outer_dock_target(outer, context->IO.MousePos);
    ImGuiWindowClass blocked;
    blocked.ClassId = ImHashStr("##outer-dock-target-only");
    blocked.DockingAllowUnclassed = false;
    // The central node is the live viewport and must remain open for rendering and picks (#4069).
    return ImGui::DockSpace(id, ImVec2(width, height),
                           ImGuiDockNodeFlags_PassthruCentralNode | ImGuiDockNodeFlags_NoDockingInCentralNode,
                           route_outer ? &blocked : nullptr);
}

void oblikovati_gui_prepare_dock_hit_test(float x, float y) {
    ImGuiContext* context = ImGui::GetCurrentContext();
    const ImGuiID id = workspace_dockspace_id();
    ImGuiDockNode* node = ImGui::DockBuilderGetNode(id);
    ImGuiWindow* outer = ImGui::FindWindowByName("##workspace-outer-host");
    if (!node || !node->HostWindow || !node->HostWindow->ParentWindow || !outer) return;
    const bool route_outer = context->MovingWindow && is_outer_dock_target(outer, ImVec2(x, y));
    ImGuiWindow* inner = node->HostWindow;
    ImGuiWindow* parent = inner->ParentWindow;
    if (route_outer) {
        inner->Flags |= ImGuiWindowFlags_NoMouseInputs;
        parent->Flags |= ImGuiWindowFlags_NoMouseInputs;
    } else {
        inner->Flags &= ~ImGuiWindowFlags_NoMouseInputs;
        parent->Flags &= ~ImGuiWindowFlags_NoMouseInputs;
    }
}

std::uint32_t oblikovati_gui_outer_dockspace(float width, float height, bool seed_asset_dock) {
    const ImGuiID id = ImGui::GetID("##outer-dockspace");
    if (!ImGui::DockBuilderGetNode(id)) {
        ImGui::DockBuilderAddNode(id, ImGuiDockNodeFlags_DockSpace);
        ImGui::DockBuilderSetNodeSize(id, ImVec2(width, height));
        ImGuiID center = id;
        const ImGuiID left = ImGui::DockBuilderSplitNode(center, ImGuiDir_Left, 0.25f, nullptr, &center);
        ImGui::DockBuilderDockWindow("Model", left);
        ImGui::DockBuilderDockWindow("Log", left);
        ImGui::DockBuilderDockWindow("Favorites", left);
        if (seed_asset_dock) {
            const ImGuiID right = ImGui::DockBuilderSplitNode(center, ImGuiDir_Right, 0.30f, nullptr, &center);
            ImGui::DockBuilderDockWindow("Material Browser", right);
            ImGui::DockBuilderDockWindow("Appearance Browser", right);
        }
        ImGui::DockBuilderFinish(id);
    }
    ImVector<CompactDockWidth> compacted;
    compact_outer_docks(ImGui::DockBuilderGetCentralNode(id), width, compacted);
    const ImGuiID dockspace = ImGui::DockSpace(id, ImVec2(width, height),
        ImGuiDockNodeFlags_PassthruCentralNode | ImGuiDockNodeFlags_NoDockingInCentralNode);
    restore_outer_dock_widths(compacted);
    return dockspace;
}

void oblikovati_gui_ensure_tabbed_window(const char* title, const char* group) {
    if (ImGui::FindWindowByName(title)) return;
    if (ImGui::FindWindowSettings(ImHashStr(title))) return;
    ImGuiWindow* group_window = ImGui::FindWindowByName(group);
    if (!group_window || !group_window->DockNode) return;
    ImGui::DockBuilderDockWindow(title, group_window->DockNode->ID);
}

bool oblikovati_gui_outer_bottom_span(std::uint32_t id, float* left, float* right) {
    ImGuiDockNode* root = ImGui::DockBuilderGetNode(id);
    ImGuiDockNode* node = ImGui::DockBuilderGetCentralNode(id);
    if (!root || !node) return false;
    ImGuiDockNode* bottom = nullptr;
    while (node->ParentNode) {
        ImGuiDockNode* parent = node->ParentNode;
        if (parent->SplitAxis == ImGuiAxis_Y && parent->ChildNodes[0] == node && parent->ChildNodes[1])
            bottom = parent->ChildNodes[1];
        node = parent;
    }
    if (!bottom || !bottom->IsVisible || bottom->Size.x <= 0) return false;
    *left = bottom->Pos.x;
    *right = bottom->Pos.x + bottom->Size.x;
    return true;
}

bool oblikovati_gui_dockspace_center(std::uint32_t id, float* left, float* top, float* right, float* bottom) {
    const ImGuiDockNode* center = ImGui::DockBuilderGetCentralNode(id);
    if (!center || center->Size.x <= 0 || center->Size.y <= 0) return false;
    *left = center->Pos.x;
    *top = center->Pos.y;
    *right = center->Pos.x + center->Size.x;
    *bottom = center->Pos.y + center->Size.y;
    return true;
}

bool oblikovati_gui_icon_button(const unsigned char* bytes, std::uint32_t texture,
                                float width, float height, float icon_width, float icon_height, bool large, bool show_caption) {
    const char* label = reinterpret_cast<const char*>(bytes);
    // Normal Button owns identity, navigation, disabled state, hit testing and last-item state.
    ImGui::PushStyleColor(ImGuiCol_Text, ImVec4(0,0,0,0));
    const bool clicked = ImGui::Button(label, ImVec2(width, height));
    ImGui::PopStyleColor();
    if (ImGui::IsItemVisible()) draw_ribbon_icon(label, texture, ImVec2(icon_width, icon_height), large, show_caption);
    return clicked;
}

void oblikovati_gui_fit_dialog(const unsigned char* title, float x, float y, float width, float height,
                              bool reset_size_on_reopen) {
    const char* name = reinterpret_cast<const char*>(title);
    const ImGuiWindow* previous = ImGui::FindWindowByName(name);
    // ImGui applies saved docking and placement when the window is first created (#4069).
    if (!previous && ImGui::FindWindowSettings(ImHashStr(name)))
        return;
    // Dock nodes own their members' bounds; fitting an individual member would undo that layout.
    if (previous && previous->DockId != 0)
        return;
    const ImVec2 position = previous ? previous->Pos : ImVec2(x, y);
    const bool keep_previous_size = previous && (!reset_size_on_reopen || previous->WasActive);
    const ImVec2 size = keep_previous_size ? previous->SizeFull : ImVec2(width, height);
    const ImVec2 display = ImGui::GetIO().DisplaySize;
    const ImVec2 fitted_size(ImClamp(size.x, 1.0f, ImMax(1.0f, display.x - 32.0f)),
                             ImClamp(size.y, 1.0f, ImMax(1.0f, display.y - 32.0f)));
    const ImVec2 fitted_position(ImClamp(position.x, 0.0f, ImMax(0.0f, display.x - fitted_size.x - 16.0f)),
                                 ImClamp(position.y, 0.0f, ImMax(0.0f, display.y - fitted_size.y - 16.0f)));
    if (!previous || position.x != fitted_position.x || position.y != fitted_position.y)
        ImGui::SetNextWindowPos(fitted_position, ImGuiCond_Always);
    if (!previous || previous->SizeFull.x != fitted_size.x || previous->SizeFull.y != fitted_size.y)
        ImGui::SetNextWindowSize(fitted_size, ImGuiCond_Always);
}

void oblikovati_gui_window_position(float x, float y, int condition) {
    ImGui::SetNextWindowPos(ImVec2(x, y), condition);
}
void oblikovati_gui_window_size(float width, float height, int condition) {
    ImGui::SetNextWindowSize(ImVec2(width, height), condition);
}
void oblikovati_gui_cursor_position(float x, float y) {
    ImGui::SetCursorPos(ImVec2(x, y));
}
bool oblikovati_gui_button(const unsigned char* label, float width, float height, bool selected) {
    if (selected) ImGui::PushStyleColor(ImGuiCol_Button, ImGui::GetStyleColorVec4(ImGuiCol_ButtonActive));
    const bool clicked = ImGui::Button(reinterpret_cast<const char*>(label), ImVec2(width, height));
    if (selected) ImGui::PopStyleColor();
    return clicked;
}
bool oblikovati_gui_begin_scroll_region(const unsigned char* identity, float width, float height) {
    return ImGui::BeginChild(reinterpret_cast<const char*>(identity), ImVec2(width, height), false,
                             ImGuiWindowFlags_HorizontalScrollbar);
}
void oblikovati_gui_dummy(float width, float height) {
    ImGui::Dummy(ImVec2(width, height));
}
bool oblikovati_gui_selectable(const unsigned char* label, bool selected, int flags) {
    return ImGui::Selectable(reinterpret_cast<const char*>(label), selected, flags, ImVec2(0, 0));
}
void oblikovati_gui_literal_text(const unsigned char* text, std::size_t length,
                                float x, float y, unsigned int color, bool background) {
    const char* begin = reinterpret_cast<const char*>(text);
    // #3839: annotations need the same scalar ABI even before a window is active.
    ImDrawList* draw = background ? ImGui::GetBackgroundDrawList() : ImGui::GetWindowDrawList();
    draw->AddText(ImVec2(x, y), color, begin, begin + length);
}
std::uint32_t oblikovati_gui_draw_dock_tab_actions() {
    ImGuiContext* context = ImGui::GetCurrentContext();
    if (!context) return 0;
    std::uint32_t actions = 0;
    for (int index = 0; index < context->DockContext.Nodes.Data.Size; ++index) {
        ImGuiDockNode* node = static_cast<ImGuiDockNode*>(context->DockContext.Nodes.Data[index].val_p);
        if (!node || !node->TabBar || node->TabBar->CurrFrameVisible != context->FrameCount)
            continue;
        // A tab bar can stay current after its host is clipped. ImGui cannot amend
        // tab items in a host whose Begin call skips its contents.
        if (!node->HostWindow || node->HostWindow->SkipItems) continue;
        if (!ImGui::DockNodeBeginAmendTabBar(node)) continue;

        const ImGuiTabItemFlags trailing = ImGuiTabItemFlags_Trailing;
        if (ImGui::TabItemButton("   ##dock-panel-menu", trailing))
            ImGui::OpenPopup("##dock-panel-menu-popup");
        const ImVec2 menu_min = ImGui::GetItemRectMin();
        const ImVec2 menu_max = ImGui::GetItemRectMax();
        const ImVec2 menu_center((menu_min.x + menu_max.x) * 0.5f, (menu_min.y + menu_max.y) * 0.5f);
        const float menu_extent = ImGui::GetFontSize() * 0.18f;
        ImDrawList* draw = ImGui::GetWindowDrawList();
        const ImU32 icon_color = ImGui::GetColorU32(ImGuiCol_Text);
        draw->AddLine(ImVec2(menu_center.x - menu_extent, menu_center.y),
                      ImVec2(menu_center.x + menu_extent, menu_center.y), icon_color, 1.5f);
        draw->AddLine(ImVec2(menu_center.x, menu_center.y - menu_extent),
                      ImVec2(menu_center.x, menu_center.y + menu_extent), icon_color, 1.5f);
        if (ImGui::IsItemHovered()) ImGui::SetTooltip("Open dockable windows");
        actions |= dock_panel_menu();

        if (ImGui::TabItemButton("   ##dock-search", trailing)) actions |= 1u << 8;
        const ImVec2 search_min = ImGui::GetItemRectMin();
        const ImVec2 search_max = ImGui::GetItemRectMax();
        const ImVec2 search_center((search_min.x + search_max.x) * 0.5f,
                                   (search_min.y + search_max.y) * 0.5f);
        const float lens = ImGui::GetFontSize() * 0.18f;
        draw->AddCircle(ImVec2(search_center.x - lens * 0.2f, search_center.y - lens * 0.2f),
                        lens, icon_color, 12, 1.5f);
        draw->AddLine(ImVec2(search_center.x + lens * 0.45f, search_center.y + lens * 0.45f),
                      ImVec2(search_center.x + lens, search_center.y + lens), icon_color, 1.5f);
        if (ImGui::IsItemHovered()) ImGui::SetTooltip("Find in Model Browser");
        ImGui::DockNodeEndAmendTabBar();
    }
    if (actions & (1u << 8)) actions = 1u << 8;
    return actions;
}

std::uint32_t oblikovati_gui_draw_workspace_panel_menu() {
    if (ImGui::Button("+##workspace-panel-add")) ImGui::OpenPopup("##dock-panel-menu-popup");
    if (ImGui::IsItemHovered()) ImGui::SetTooltip("Open dockable windows");
    return dock_panel_menu();
}
}
