#include <TGUI/TGUI.hpp>
#include <TGUI/Backend/SFML-Graphics.hpp>
#include <iostream>
#include <httplib.h>
#include <fstream>
#include <nlohmann/json.hpp>

class LocalStorage {
public:
    bool SetTokens(std::string access_token, std::string refresh_token) {
        // �������� ����� ��� ������
        std::ofstream out_file("localStorage.txt");

        // ��������, ������ �� ����
        if (!out_file.is_open()) {
            std::cerr << "������ �������� �����!" << std::endl;
            return 1;
        }

        // ������ ������ � ����
        out_file << "access_token: " << access_token << std::endl;
        out_file << "refresh_token: " << refresh_token << std::endl;

        // �������� �����
        out_file.close();

        std::cout << "������ �������� � ����." << std::endl;
        return 0;
    }

    void Clear() {
        // �������� ����� ��� ������
        std::ofstream out_file("localStorage.txt");

        // ��������, ������ �� ����
        if (!out_file.is_open()) {
            std::cerr << "������ �������� �����!" << std::endl;
            return;
        }

        // ������ ������ � ����
        out_file << "access_token: " << std::endl;
        out_file << "refresh_token: " << std::endl;

        // �������� �����
        out_file.close();

        std::cout << "������ �������� � ����." << std::endl;
    }

    std::string GetAccessToken() {
        std::ifstream in("localStorage.txt");

        // ��������, ������ �� ����
        if (!in.is_open()) {
            std::cerr << "������ �������� �����!" << std::endl;
            return "";
        }

        std::string line;
        while (std::getline(in, line)) {
            if (line.find("access_token: ") == 0) {
                in.close();
                return line.substr(14); // ������� "access_token: " (14 ��������)
            }
        }
    }

    std::string GetRefreshToken() {
        std::ifstream in("localStorage.txt");

        // ��������, ������ �� ����
        if (!in.is_open()) {
            std::cerr << "������ �������� �����!" << std::endl;
            return "";
        }

        std::string line;
        while (std::getline(in, line)) {
            if (line.find("refresh_token: ") == 0) {
                in.close();
                return line.substr(15); // ������� "access_token: " (14 ��������)
            }
        }
    }
};

void login();
void exit();
bool verifyToken(const std::string &str_token);
std::string removeChar(std::string str, char ch);

tgui::EditBox::Ptr editBoxLogin;
tgui::EditBox::Ptr editBoxPassword;

LocalStorage localStorage;

int CurrentPage = 0;
// 0 - �����������
// 1 - ������ ������
// 2 - ����

bool authorized = 0;

int main()
{
    setlocale(LC_ALL, "ru");

    sf::RenderWindow window{ {1280, 720}, L"�������� ������" };
    tgui::Gui gui{ window };

    try {
        tgui::Font font("Inter_18pt-Regular.ttf");
        gui.setFont(font);
    }
    catch (const tgui::Exception& e) {
        std::cerr << "Failed to load font: " << e.what() << std::endl;
        return 1;
    }

    std::string token = localStorage.GetAccessToken();

    if (verifyToken(token)) {
        CurrentPage = 1;
        tgui::Label::Ptr label = tgui::Label::create(u8"��������� �����");
        label->setPosition({ 25, 25 });
        label->setTextSize(18);

        tgui::Button::Ptr exitButton = tgui::Button::create(u8"�����");
        exitButton->setPosition({ 25, 650 });
        exitButton->setSize({ 60, 25 });
        exitButton->setTextSize(14);
        exitButton->onClick([]() {
            exit();
            });

        auto listView = tgui::ListView::create();
        listView->setSize(650, 500);
        listView->setPosition(25, 60);

        // ���������� �������
        listView->addColumn(u8"��������", 250);
        listView->addColumn(u8"������ ���� ��������� ��", 200);
        listView->addColumn(u8"����������� �������", 200);

        // ���������� �����
        listView->addItem({ u8"������ ����������", "25.03.2025", u8"1 ���" });
        listView->addItem({ u8"���������� ����������", "31.03.2025", u8"15 �����" });
        listView->addItem({ u8"������ ������������ � �������������� ����������", "28.03.2025", u8"���" });

        // ��������� ������������ ������
        listView->setColumnAlignment(1, tgui::HorizontalAlignment::Center);


        gui.add(listView);

        gui.add(label);
        gui.add(exitButton);
    }
    else {
        // GUI
        tgui::Label::Ptr label = tgui::Label::create(u8"�����������");
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

        tgui::Button::Ptr button = tgui::Button::create(u8"�����");
        button->setPosition({ 610, 330 });
        button->setSize({ 60, 25 });
        button->setTextSize(14);
        button->onClick(login);

        gui.add(label);
        gui.add(button);
        gui.add(editBoxLogin, "login");
        gui.add(editBoxPassword, "password");
        // GUI
    }

    while (window.isOpen())
    {
        sf::Event event;
        while (window.pollEvent(event))
        {
            gui.handleEvent(event);

            if (event.type == sf::Event::Closed)
                window.close();
        }

        if (!authorized && CurrentPage != 0) {
            gui.removeAllWidgets();

            tgui::Label::Ptr label = tgui::Label::create(u8"�����������");
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

            tgui::Button::Ptr button = tgui::Button::create(u8"�����");
            button->setPosition({ 610, 330 });
            button->setSize({ 60, 25 });
            button->setTextSize(14);
            button->onClick(login);

            gui.add(label);
            gui.add(button);
            gui.add(editBoxLogin, "login");
            gui.add(editBoxPassword, "password");
            CurrentPage = 0;
        }

        if (authorized && CurrentPage != 1) {
            CurrentPage = 1;
            gui.removeAllWidgets();

            tgui::Label::Ptr label = tgui::Label::create(u8"��������� �����");
            label->setPosition({ 25, 25 });
            label->setTextSize(18);

            tgui::Button::Ptr exitButton = tgui::Button::create(u8"�����");
            exitButton->setPosition({ 25, 650 });
            exitButton->setSize({ 60, 25 });
            exitButton->setTextSize(14);
            exitButton->onClick([]() {
                exit();
                });

            auto listView = tgui::ListView::create();
            listView->setSize(650, 500);
            listView->setPosition(25, 60);

            // ���������� �������
            listView->addColumn(u8"��������", 150);
            listView->addColumn(u8"������ ���� ��������� ��", 200);
            listView->addColumn(u8"����������� �������", 150);

            // ���������� �����
            listView->addItem({ u8"������ ����������", "25.03.2025", u8"1 ���" });
            listView->addItem({ u8"���������� ����������", "31.03.2025", u8"15 �����" });
            listView->addItem({ u8"������ ������������ � �������������� ����������", "28.03.2025", u8"���" });

            // ��������� ������������ ������
            listView->setColumnAlignment(1, tgui::HorizontalAlignment::Center);


            gui.add(listView);

            gui.add(label);
            gui.add(exitButton);
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

static bool verifyToken(const std::string& str_token) {
    if (str_token == "")
        return 0;

    httplib::Client cli("localhost:8080");

    // POST-������ � JSON
    httplib::Headers headers = {
        {"Authorization", str_token},
        {"Content-Type", "application/json"}
    };
    
    auto post_res = cli.Post("/api/verify", headers, "", "application/json");
    if (post_res && post_res->status == 200) {
        std::cout << "Token verified" << std::endl;
        authorized = true;
        return true;
    }
    else {
        headers = {
            {"Authorization", localStorage.GetRefreshToken()},
            {"Content-Type", "application/json"}
        };

        auto post_res_refresh = cli.Post("/api/refreshtoken", headers, "", "application/json");
        if (post_res_refresh && post_res_refresh->status == 200) {
            std::string access_token = removeChar(post_res_refresh->body, '"');
            localStorage.SetTokens(access_token, localStorage.GetRefreshToken());
            authorized = true;
            return true;
        }
        else {
            localStorage.Clear();
            authorized = false;
            return false;
        }
        authorized = false;
        return false;
    }
}

std::string removeChar(std::string str, char ch) {
    // ���������� �������� erase-remove ��� �������� �������
    str.erase(std::remove(str.begin(), str.end(), ch), str.end());
    return str;
}

void login() {
    if (validate()) {
        std::cout << "Validated" << std::endl;
    }
    httplib::Client cli("localhost:8080");

    // POST-������ � JSON
    httplib::Headers headers = {
        {"Content-Type", "application/json"}
    };
    std::string body = R"({"username": ")" + editBoxLogin->getText().toStdString() + R"(", "password": ")" + editBoxPassword->getText().toStdString() + R"("})";

    auto post_res = cli.Post("/api/login", headers, body, "application/json");
    if (post_res && post_res->status == 200) {
        // ������ � ��������� ������
        nlohmann::json j = nlohmann::json::parse(post_res->body);

        std::string access_token = j["access_token"].get<std::string>();
        std::string refresh_token = j["refresh_token"].get<std::string>();
        
        localStorage.SetTokens(access_token, refresh_token);
        authorized = true;
    }
    else {
        std::cout << "Invalid credentials" << std::endl;
    }
}

void exit() {
    localStorage.Clear();
    authorized = false;
}

void getTestsData() {
    if (!authorized) {
        return;
    }

    httplib::Client cli("localhost:8080");

    // POST-������ � JSON
    httplib::Headers headers = {
        {"Authorization", localStorage.GetAccessToken()},
        {"Content-Type", "application/json"}
    };
    
    auto post_res = cli.Post("/api/gettestsdata", headers, "", "application/json");

    if (post_res && post_res->status == 200) {

    }
}