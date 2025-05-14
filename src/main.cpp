#include <TGUI/TGUI.hpp>
#include <TGUI/Backend/SFML-Graphics.hpp>
#include <iostream>
#include <httplib.h>

void login();

tgui::EditBox::Ptr editBoxLogin;
tgui::EditBox::Ptr editBoxPassword;

int main()
{
    setlocale(LC_ALL, "ru");

    sf::RenderWindow window{ {1280, 720}, L"Курсовая работа" };
    tgui::Gui gui{ window };

    try {
        tgui::Font font("Inter_18pt-Regular.ttf");
        gui.setFont(font);
    }
    catch (const tgui::Exception& e) {
        std::cerr << "Failed to load font: " << e.what() << std::endl;
        return 1;
    }

    // GUI
    tgui::Label::Ptr label = tgui::Label::create(u8"Авторизация");
    label->setPosition({ 560, 200 });
    label->setTextSize(18);

    editBoxLogin = tgui::EditBox::create();
    editBoxLogin->setPosition({ 560, 250 });
    editBoxLogin->setSize({ 160, 20 });
    editBoxLogin->setTextSize(14);

    editBoxPassword = tgui::EditBox::create();
    editBoxPassword->setPosition({ 560, 290 });
    editBoxPassword->setSize({ 160, 20 });
    editBoxPassword->setTextSize(14);

    tgui::Button::Ptr button = tgui::Button::create(u8"Войти");
    button->setPosition({ 610, 330 });
    button->setSize({ 60, 25 });
    button->setTextSize(14);
    button->onClick(login);

    gui.add(label);
    gui.add(button);
    gui.add(editBoxLogin, "login");
    gui.add(editBoxPassword, "password");
    // GUI

    while (window.isOpen())
    {
        sf::Event event;
        while (window.pollEvent(event))
        {
            gui.handleEvent(event);

            if (event.type == sf::Event::Closed)
                window.close();
        }

        window.clear(sf::Color(255, 255, 255, 255));

        gui.draw();

        window.display();
    }
}

static bool validate() {
    if (editBoxLogin->getText() != "" && editBoxPassword->getText() != "")
        return true;
    return false;
}

void login() {
    if (validate()) {
        std::cout << "Validated" << std::endl;
    }
    httplib::Client cli("localhost:8080");

    // POST-запрос с JSON
    httplib::Headers headers = {
        {"Content-Type", "application/json"}
    };
    std::string body = R"({"username": ")" + editBoxLogin->getText().toStdString() + R"(", "password": ")" + editBoxPassword->getText().toStdString() + R"("})";

    auto post_res = cli.Post("/api/login", headers, body, "application/json");
    if (post_res && post_res->status == 200) {
        std::cout << "POST Response: " << post_res->body << std::endl;
    }
}