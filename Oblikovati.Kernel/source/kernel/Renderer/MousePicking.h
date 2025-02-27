#pragma once

#include <vulkan/vulkan.h>
#include <glm/glm.hpp>

namespace Oblikovati::Kernel::VulkanRenderer
{

// Forward declarations
class Renderer;
class Buffer;

// Element types that can be picked
enum class PickableElementType {
    None,
    Vertex,
    Edge,
    Face
};

// Element ID structure
struct PickableElementId {
    PickableElementType type;
    uint32_t modelId;   // ID of the model
    uint32_t meshId;    // ID of the mesh within the model
    uint32_t elementId; // ID of the element (vertex/edge/face) within the mesh
    
    bool operator==(const PickableElementId& other) const {
        return type == other.type && 
               modelId == other.modelId && 
               meshId == other.meshId && 
               elementId == other.elementId;
    }
    
    bool operator!=(const PickableElementId& other) const {
        return !(*this == other);
    }
    
    // For use as key in hash maps
    struct Hash {
        std::size_t operator()(const PickableElementId& id) const {
            return std::hash<uint32_t>()((static_cast<uint32_t>(id.type) << 30) ^ 
                   (id.modelId << 20) ^ 
                   (id.meshId << 10) ^ 
                   id.elementId);
        }
    };
};

// Callback type for picking events
using PickingCallback = std::function<void(const PickableElementId& pickedElement)>;

class MousePicking {
public:
    MousePicking(VkDevice device, VkPhysicalDevice physicalDevice, Window* window, Camera* camera);
    ~MousePicking();
    
    // Prevent copying
    MousePicking(const MousePicking&) = delete;
    MousePicking& operator=(const MousePicking&) = delete;
    
    // Initialize the picking resources
    void init(Renderer* renderer, uint32_t width, uint32_t height);
    
    // Update the picking framebuffer
    void updateResources(uint32_t width, uint32_t height);
    
    // Setup for a picking render pass
    void beginPickingRenderPass(VkCommandBuffer commandBuffer);
    void endPickingRenderPass(VkCommandBuffer commandBuffer);
    
    // Process mouse click at screen coordinates
    PickableElementId processMouseClick(int x, int y);
    
    // Simulate mouse click for testing
    PickableElementId simulateMouseClick(int x, int y);
    
    // Register callback for picking events
    void registerPickingCallback(PickingCallback callback);
    
    // Element ID management
    PickableElementId registerVertex(uint32_t modelId, uint32_t meshId, uint32_t vertexId);
    PickableElementId registerEdge(uint32_t modelId, uint32_t meshId, uint32_t edgeId);
    PickableElementId registerFace(uint32_t modelId, uint32_t meshId, uint32_t faceId);
    
    // Get color for a specific element ID (for rendering to picking buffer)
    glm::vec4 getColorForElementId(const PickableElementId& id);
    
    // For accessing the picking pipeline
    VkPipeline getPickingPipeline() const { return m_pickingPipeline; }
    VkPipelineLayout getPickingPipelineLayout() const { return m_pickingPipelineLayout; }
    VkRenderPass getPickingRenderPass() const { return m_pickingRenderPass; }
    
    // Get the framebuffer for the current swapchain image
    VkFramebuffer getPickingFramebuffer() const { return m_pickingFramebuffer; }
    
private:
    // Initialize picking resources
    void createPickingRenderPass();
    void createPickingFramebuffer(uint32_t width, uint32_t height);
    void createPickingPipeline();
    void createPickingDescriptorSetLayout();
    
    // Cleanup resources
    void cleanup();
    
    // Convert ID to color and vice versa
    glm::uvec4 idToColor(uint32_t id);
    uint32_t colorToId(const glm::uvec4& color);
    
    // Read pixel from framebuffer
    glm::uvec4 readPixelFromFramebuffer(int x, int y);
    
    // Device references
    VkDevice m_device;
    VkPhysicalDevice m_physicalDevice;
    
    // Picking resources
    VkRenderPass m_pickingRenderPass;
    VkFramebuffer m_pickingFramebuffer;
    VkPipeline m_pickingPipeline;
    VkPipelineLayout m_pickingPipelineLayout;
    VkDescriptorSetLayout m_pickingDescriptorSetLayout;
    
    // Color attachment for picking
    VkImage m_pickingColorImage;
    VkDeviceMemory m_pickingColorImageMemory;
    VkImageView m_pickingColorImageView;
    
    // Depth attachment for picking
    VkImage m_pickingDepthImage;
    VkDeviceMemory m_pickingDepthImageMemory;
    VkImageView m_pickingDepthImageView;
    
    // Staging resources for reading back pixels
    std::unique_ptr<Buffer> m_stagingBuffer;
    VkCommandPool m_pickingCommandPool;
    
    // Mappings from IDs to elements and back
    std::unordered_map<uint32_t, PickableElementId> m_idToElement;
    std::unordered_map<PickableElementId, uint32_t, PickableElementId::Hash> m_elementToId;
    uint32_t m_nextId = 1; // Start from 1, 0 is reserved for "no selection"
    
    // Callback for picking events
    PickingCallback m_pickingCallback;
    
    // References to other components (not owned)
    Window* m_window;
    Camera* m_camera;
    Renderer* m_renderer;
};

}
