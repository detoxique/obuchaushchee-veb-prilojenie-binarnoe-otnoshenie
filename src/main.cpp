#include <TGUI/TGUI.hpp>
#include <TGUI/Backend/SFML-Graphics.hpp>

bool runExample(tgui::BackendGui& gui);

int main()
{
#ifdef TGUI_SYSTEM_IOS
    sf::RenderWindow window(sf::VideoMode::getDesktopMode(), "TGUI example (SFML-Graphics)");
#elif SFML_VERSION_MAJOR >= 3
    sf::RenderWindow window(sf::VideoMode{ {800, 600} }, "TGUI example (SFML-Graphics)");
#else
    sf::RenderWindow window({ 800, 600 }, "TGUI example (SFML-Graphics)");
#endif

    tgui::Gui gui(window);
    if (!runExample(gui))
        return EXIT_FAILURE;

    gui.mainLoop(); // To use your own main loop, see https://tgui.eu/tutorials/latest-stable/backend-sfml-graphics/
    return EXIT_SUCCESS;
}