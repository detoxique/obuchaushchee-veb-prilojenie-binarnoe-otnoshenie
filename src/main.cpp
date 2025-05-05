#include "GUI.h"
#include <windows.h>

int main()
{
    setlocale(LC_ALL, "Rus");
    auto window = sf::RenderWindow(sf::VideoMode({1280u, 720u}), "Binary Relation");
    window.setFramerateLimit(60);

    // GUI
    const sf::Font font("arial.ttf");
    sf::String btnText = L"Войти";
    Button btn({600, 345}, font, btnText);

    while (window.isOpen())
    {
        while (const std::optional event = window.pollEvent())
        {
            if (event->is<sf::Event::Closed>())
            {
                window.close();
            }
        }

        window.clear(sf::Color(206, 206, 206, 255));

        // drawing
        window.draw(btn);

        window.display();
    }
}
