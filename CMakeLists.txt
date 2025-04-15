cmake_minimum_required(VERSION 3.13 FATAL_ERROR)

set(PROJECT_NAME DesktopApp)
set(IMGUI_DIR external/imgui)

project(${PROJECT_NAME})

add_executable(${PROJECT_NAME} src/main.cpp ${IMGUI_DIR}/imgui.cpp ${IMGUI_DIR}/imgui.h ${IMGUI_DIR}/imgui_demo.cpp ${IMGUI_DIR}/imgui_draw.cpp ${IMGUI_DIR}/imgui_tables.cpp ${IMGUI_DIR}/imgui_widgets.cpp ${IMGUI_DIR}/backends/imgui_impl_glfw.cpp ${IMGUI_DIR}/backends/imgui_impl_glfw.h ${IMGUI_DIR}/backends/imgui_impl_opengl3.cpp ${IMGUI_DIR}/backends/imgui_impl_opengl3.h )

set(GLFW_BUILD_DOCS OFF CACHE BOOL "" FORCE)
set(GLFW_BUILD_TESTS OFF CACHE BOOL "" FORCE)
set(GLFW_BUILD_EXAMPLES OFF CACHE BOOL "" FORCE)
set(GLFW_INSTALL OFF CACHE BOOL "" FORCE)

add_subdirectory(external/glfw)
target_link_libraries(${PROJECT_NAME} glfw)
target_include_directories(${PROJECT_NAME} PUBLIC ${IMGUI_DIR})