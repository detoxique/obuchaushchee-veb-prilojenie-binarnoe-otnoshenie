cmake_minimum_required(VERSION 3.15)
project(MyTGUIApp)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)

# Путь к TGUI (укажите свой путь!)
set(TGUI_ROOT "TGUI/")

# Include Directories
include_directories(${TGUI_ROOT}/include)

# Find SFML (TGUI зависит от SFML)
include(FetchContent)
FetchContent_Declare(SFML
    GIT_REPOSITORY https://github.com/SFML/SFML.git
    GIT_TAG 3.0.1
    GIT_SHALLOW ON
    EXCLUDE_FROM_ALL
    SYSTEM)
FetchContent_MakeAvailable(SFML)

# Создаем исполняемый файл
add_executable(MyTGUIApp src/main.cpp)

# Link libraries
target_link_libraries(MyTGUIApp sfml-graphics sfml-window sfml-system tgui)