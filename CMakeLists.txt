cmake_minimum_required(VERSION 3.13 FATAL_ERROR)

include(FetchContent)

set(PROJECT_NAME DesktopApp)
# Set properties that will be used while building TGUI
set(BUILD_SHARED_LIBS OFF)      # Determines whether TGUI build a static or dynamic/shared library
set(TGUI_BACKEND SFML_GRAPHICS)  # Sets which backend TGUI will use

FetchContent_Declare(
  TGUI
  GIT_REPOSITORY https://github.com/texus/TGUI
  GIT_TAG v1.8.0  # Change this to wanted version
)
FetchContent_MakeAvailable(TGUI)

project(${PROJECT_NAME})

add_executable(${PROJECT_NAME} src/main.cpp)

target_link_libraries(${PROJECT_NAME} PRIVATE TGUI::TGUI)