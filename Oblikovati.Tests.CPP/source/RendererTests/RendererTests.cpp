#include <gtest/gtest.h>
#include "kernel/Renderer/Model.h"
#include "../OblikovatiApplicationTestHost.h"
#include <memory>
#include <string>
#include <glm/glm/vec3.hpp>

#include "kernel/Application.h"

namespace Oblikovati::Host
{
	class OblikovatiApplicationTestHost;
}

namespace Oblikovati::RendererTests
{
	class RendererTest : public ::testing::Test
	{
	protected:
		void SetUp() override
		{
			Kernel::ApplicationConfiguration config;

			config.Width = 800;
			config.Height = 600;
			config.Title = "Renderer Test";
		
			// Create application with test settings
			m_app = std::make_unique<Host::OblikovatiApplicationTestHost>(config);
		}

		void TearDown() override
		{
			m_app.reset();
		}

		std::unique_ptr<Host::OblikovatiApplicationTestHost> m_app;

		// Helper function to create a simple triangle model
		static std::unique_ptr<Kernel::VulkanRenderer::Model> createTriangleModel()
		{
			std::vector<Kernel::VulkanRenderer::Vertex> vertices = {
				{{-0.5f, -0.5f, 0.0f}, {0.0f, 0.0f, 1.0f}, {0.0f, 0.0f}, {1.0f, 0.0f, 0.0f}},
				{{0.5f, -0.5f, 0.0f}, {0.0f, 0.0f, 1.0f}, {1.0f, 0.0f}, {0.0f, 1.0f, 0.0f}},
				{{0.0f, 0.5f, 0.0f}, {0.0f, 0.0f, 1.0f}, {0.5f, 1.0f}, {0.0f, 0.0f, 1.0f}}
			};

			std::vector<uint32_t> indices = { 0, 1, 2 };
			std::vector<glm::vec3> normals = { {0.0f, 0.0f, 1.0f} };

			auto model = std::make_unique<Kernel::VulkanRenderer::Model>();
			model->loadFromData(vertices, indices, normals);
			return model;
		}

		// Helper function to create a simple cube model
		static std::unique_ptr<Kernel::VulkanRenderer::Model> createCubeModel()
		{
			// Define cube vertices (8 corners)
			std::vector<Kernel::VulkanRenderer::Vertex> vertices = {
				// Front face
				{{-0.5f, -0.5f, 0.5f}, {0.0f, 0.0f, 1.0f}, {0.0f, 0.0f}, {1.0f, 0.0f, 0.0f}},  // 0: Bottom-left
				{{0.5f, -0.5f, 0.5f}, {0.0f, 0.0f, 1.0f}, {1.0f, 0.0f}, {0.0f, 1.0f, 0.0f}},   // 1: Bottom-right
				{{0.5f, 0.5f, 0.5f}, {0.0f, 0.0f, 1.0f}, {1.0f, 1.0f}, {0.0f, 0.0f, 1.0f}},    // 2: Top-right
				{{-0.5f, 0.5f, 0.5f}, {0.0f, 0.0f, 1.0f}, {0.0f, 1.0f}, {1.0f, 1.0f, 0.0f}},   // 3: Top-left

				// Back face
				{{-0.5f, -0.5f, -0.5f}, {0.0f, 0.0f, -1.0f}, {1.0f, 0.0f}, {1.0f, 0.0f, 1.0f}},  // 4: Bottom-left
				{{0.5f, -0.5f, -0.5f}, {0.0f, 0.0f, -1.0f}, {0.0f, 0.0f}, {0.0f, 1.0f, 1.0f}},   // 5: Bottom-right
				{{0.5f, 0.5f, -0.5f}, {0.0f, 0.0f, -1.0f}, {0.0f, 1.0f}, {1.0f, 0.0f, 1.0f}},    // 6: Top-right
				{{-0.5f, 0.5f, -0.5f}, {0.0f, 0.0f, -1.0f}, {1.0f, 1.0f}, {0.5f, 0.5f, 0.5f}}    // 7: Top-left
			};

			// Define indices for 12 triangles (6 faces, 2 triangles per face)
			std::vector<uint32_t> indices = {
				// Front face
				0, 1, 2, 2, 3, 0,
				// Back face
				5, 4, 7, 7, 6, 5,
				// Left face
				4, 0, 3, 3, 7, 4,
				// Right face
				1, 5, 6, 6, 2, 1,
				// Top face
				3, 2, 6, 6, 7, 3,
				// Bottom face
				4, 5, 1, 1, 0, 4
			};

			// Define normals for each face
			std::vector<glm::vec3> normals = {
				{0.0f, 0.0f, 1.0f},   // Front
				{0.0f, 0.0f, -1.0f},  // Back
				{-1.0f, 0.0f, 0.0f},  // Left
				{1.0f, 0.0f, 0.0f},   // Right
				{0.0f, 1.0f, 0.0f},   // Top
				{0.0f, -1.0f, 0.0f}   // Bottom
			};

			auto model = std::make_unique<Kernel::VulkanRenderer::Model>();
			model->loadFromData(vertices, indices, normals);
			return model;
		}
	};

	// Test rendering a single frame with a simple triangle model
	TEST_F(RendererTest, RenderTriangle)
	{
		ASSERT_TRUE(m_app->loadModel(createTriangleModel()));
		ASSERT_TRUE(m_app->renderSingleFrame());
	}

	// Test rendering a single frame with a cube model
	TEST_F(RendererTest, RenderCube)
	{
		ASSERT_TRUE(m_app->loadModel(createCubeModel()));
		ASSERT_TRUE(m_app->renderSingleFrame());
	}

	// Test capturing a framebuffer to a file
	TEST_F(RendererTest, CaptureFramebuffer)
	{
		ASSERT_TRUE(m_app->loadModel(createCubeModel()));
		ASSERT_TRUE(m_app->renderSingleFrame());
		ASSERT_TRUE(m_app->captureFramebuffer("test_output.png"));
	}

	// Test comparing framebuffer with a reference image
	TEST_F(RendererTest, CompareWithReference)
	{
		ASSERT_TRUE(m_app->loadModel(createCubeModel()));
		ASSERT_TRUE(m_app->renderSingleFrame());

		// First capture a reference image
		ASSERT_TRUE(m_app->captureFramebuffer("reference.png"));

		// Render again and compare with reference
		ASSERT_TRUE(m_app->renderSingleFrame());
		ASSERT_TRUE(m_app->compareWithReference("reference.png", 0.01f));
	}
}
