#include <TGUI/TGUI.hpp>
#include <TGUI/Backend/SFML-Graphics.hpp>

bool runExample(tgui::BackendGui& gui)
{
    return true;
}

int main()
{
    sf::RenderWindow window{ {800, 600}, "TGUI example - SFML_GRAPHICS backend" };

    tgui::Gui gui{window};
    if (runExample(gui))
        gui.mainLoop();
}