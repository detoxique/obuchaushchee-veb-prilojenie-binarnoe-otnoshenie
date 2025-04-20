cmake_minimum_required(VERSION 3.13 FATAL_ERROR)

include(FetchContent)

set(PROJECT_NAME DesktopApp)
# Set properties that will be used while building TGUI
set(BUILD_SHARED_LIBS OFF)      # Determines whether TGUI build a static or dynamic/shared library
set(TGUI_BACKEND SFML_GRAPHICS)  # Sets which backend TGUI will use
set(SFML_INCLUDE_DIR external/sfml/include)
set(SFML_LIBRARIES external/sfml/lib)
set(SFML_DEPENDENCIES external/sfml/bin)
set(SFML_DIR external/sfml)
set(CMAKE_MODULE_PATH ${CMAKE_CURRENT_SOURCE_DIR}/cmake)

find_package(SFML REQUIRED COMPONENTS graphics window system) # Look for SFML

include_directories(${SFML_INCLUDE_DIR})

FetchContent_Declare(
  TGUI
  GIT_REPOSITORY https://github.com/texus/TGUI
  GIT_TAG v1.8.0  # Change this to wanted version
)
FetchContent_MakeAvailable(TGUI)

project(${PROJECT_NAME})

add_executable(${PROJECT_NAME} src/main.cpp)

target_link_libraries(${PROJECT_NAME} PRIVATE TGUI::TGUI)
target_link_libraries(${PROJECT_NAME} ${SFML_LIBRARIES} ${SFML_DEPENDENCIES}) # Link SFML and dependencies