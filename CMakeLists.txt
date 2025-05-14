cmake_minimum_required(VERSION 3.15)
project(MyTGUIApp)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_RUNTIME_OUTPUT_DIRECTORY ${CMAKE_BINARY_DIR}/bin)

# Include Directories
include_directories(${TGUI_ROOT}/include)

# Find SFML (TGUI зависит от SFML)
include(FetchContent)
FetchContent_Declare(SFML
    GIT_REPOSITORY https://github.com/SFML/SFML.git
    GIT_TAG 2.6.2
    GIT_SHALLOW ON
    EXCLUDE_FROM_ALL
    SYSTEM)
FetchContent_MakeAvailable(SFML)

# Загрузка и сборка TGUI
FetchContent_Declare(
    tgui
    GIT_REPOSITORY https://github.com/texus/TGUI.git
    GIT_TAG 1.x
)
FetchContent_MakeAvailable(tgui)

FetchContent_Declare(
    httplib
    GIT_REPOSITORY https://github.com/yhirose/cpp-httplib.git
)
FetchContent_MakeAvailable(httplib)



# Создаем исполняемый файл
add_executable(MyTGUIApp src/main.cpp)

# Link libraries
target_link_libraries(MyTGUIApp PRIVATE sfml-graphics sfml-window sfml-system tgui httplib)
