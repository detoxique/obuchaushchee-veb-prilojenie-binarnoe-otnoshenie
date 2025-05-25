#include <TGUI/TGUI.hpp>
#include <TGUI/Backend/SFML-Graphics.hpp>
#include <iostream>
#include <httplib.h>
#include <fstream>
#include <nlohmann/json.hpp>
#include <vector>

#if defined(_WIN32) || defined(_WIN64)
#include <windows.h>
#include <comdef.h>
#include <wbemidl.h>
#pragma comment(lib, "wbemuuid.lib")
#elif defined(__linux__) || defined(__APPLE__)
#include <unistd.h>
#include <sys/ioctl.h>
#include <net/if.h>
#include <sys/socket.h>
#endif

std::string Interface_Name = "";

std::string getInterfaceName() {
    std::string interface_name;
#if defined(_WIN32) || defined(_WIN64)
    interface_name = "Ethernet";
#elif defined(__linux__)
    // Получаем интерфейс по умолчанию через ip route
    FILE* pipe = popen("ip route show default | awk '/default/ {print $5}'", "r");
    if (pipe) {
        char buffer[128];
        if (fgets(buffer, sizeof(buffer), pipe) != nullptr) {
            // Удаляем символ новой строки
            size_t len = strlen(buffer);
            if (len > 0 && buffer[len - 1] == '\n') {
                buffer[len - 1] = '\0';
            }
            interface_name = buffer;
        }
        pclose(pipe);
}
    // Если не удалось определить, используем значение по умолчанию
    if (interface_name.empty()) {
        interface_name = "eth0";
    }
#elif defined(__APPLE__)
    // Получаем интерфейс по умолчанию через route
    FILE* pipe = popen("route get default 2>/dev/null | awk '/interface:/ {print $2}'", "r");
    if (pipe) {
        char buffer[128];
        if (fgets(buffer, sizeof(buffer), pipe) != nullptr) {
            // Удаляем символ новой строки
            size_t len = strlen(buffer);
            if (len > 0 && buffer[len - 1] == '\n') {
                buffer[len - 1] = '\0';
            }
            interface_name = buffer;
        }
        pclose(pipe);
    }
    // Если не удалось определить, используем значение по умолчанию
    if (interface_name.empty()) {
        interface_name = "en0";
    }
#endif
    return interface_name;
}

bool disable_network_interface() {
    std::string interface_name = Interface_Name;

#if defined(_WIN32) || defined(_WIN64)
    // Windows: Используем netsh (простой способ)
    std::string command = "netsh interface set interface \"" + interface_name + "\" admin=disable";
    int result = system(command.c_str());
    return (result == 0);
#elif defined(__linux__)
    // Linux: Используем ioctl
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        perror("socket");
        return false;
    }

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, interface_name.c_str(), IFNAMSIZ);

    // Получаем текущие флаги
    if (ioctl(sock, SIOCGIFFLAGS, &ifr) < 0) {
        perror("ioctl (get flags)");
        close(sock);
        return false;
    }

    // Отключаем интерфейс
    ifr.ifr_flags &= ~IFF_UP;
    if (ioctl(sock, SIOCSIFFLAGS, &ifr) < 0) {
        perror("ioctl (set flags)");
        close(sock);
        return false;
    }

    close(sock);
    return true;
#elif defined(__APPLE__)
    // macOS: Используем ifconfig (или ioctl, как в Linux)
    std::string command = "ifconfig " + interface_name + " down";
    int result = system(command.c_str());
    return (result == 0);
#else
#error "Unsupported platform"
#endif
}

bool enable_network_interface() {
    std::string interface_name = Interface_Name;

#if defined(_WIN32) || defined(_WIN64)
    // Windows: Используем netsh
    std::string command = "netsh interface set interface \"" + interface_name + "\" admin=enable";
    int result = system(command.c_str());
    return (result == 0);
#elif defined(__linux__)
    // Linux: Используем ioctl
    int sock = socket(AF_INET, SOCK_DGRAM, 0);
    if (sock < 0) {
        perror("socket");
        return false;
    }

    struct ifreq ifr;
    memset(&ifr, 0, sizeof(ifr));
    strncpy(ifr.ifr_name, interface_name.c_str(), IFNAMSIZ);

    // Получаем текущие флаги
    if (ioctl(sock, SIOCGIFFLAGS, &ifr) < 0) {
        perror("ioctl (get flags)");
        close(sock);
        return false;
    }

    // Включаем интерфейс
    ifr.ifr_flags |= IFF_UP;
    if (ioctl(sock, SIOCSIFFLAGS, &ifr) < 0) {
        perror("ioctl (set flags)");
        close(sock);
        return false;
    }

    close(sock);
    return true;
#elif defined(__APPLE__)
    // macOS: Используем ifconfig
    std::string command = "ifconfig " + interface_name + " up";
    int result = system(command.c_str());
    return (result == 0);
#else
#error "Unsupported platform"
#endif
}

class LocalStorage {
private:
    std::string acc_token = "";
public:
    bool SetTokens(std::string access_token, std::string refresh_token) {
        // Открытие файла для записи
        std::ofstream out_file("localStorage.txt");

        // Проверка, открыт ли файл
        if (!out_file.is_open()) {
            std::cerr << "Ошибка открытия файла!" << std::endl;
            return 1;
        }

        acc_token = access_token;

        // Запись строки в файл
        out_file << "access_token: " << access_token << std::endl;
        out_file << "refresh_token: " << refresh_token << std::endl;

        // Закрытие файла
        out_file.close();

        std::cout << "Строка записана в файл." << std::endl;
        return 0;
    }

    void Clear() {
        // Открытие файла для записи
        std::ofstream out_file("localStorage.txt");

        // Проверка, открыт ли файл
        if (!out_file.is_open()) {
            std::cerr << "Ошибка открытия файла!" << std::endl;
            return;
        }

        // Запись строки в файл
        out_file << "access_token: " << std::endl;
        out_file << "refresh_token: " << std::endl;

        // Закрытие файла
        out_file.close();

        std::cout << "Строка записана в файл." << std::endl;
    }

    std::string GetAccessToken() {
        if (acc_token != "")
            return acc_token;

        std::ifstream in("localStorage.txt");

        // Проверка, открыт ли файл
        if (!in.is_open()) {
            std::cerr << "Ошибка открытия файла!" << std::endl;
            return "";
        }

        std::string line;
        while (std::getline(in, line)) {
            if (line.find("access_token: ") == 0) {
                in.close();
                acc_token = line.substr(14);
                return acc_token; // Удаляем "access_token: " (14 символов)
            }
        }
    }

    std::string GetRefreshToken() {
        std::ifstream in("localStorage.txt");

        // Проверка, открыт ли файл
        if (!in.is_open()) {
            std::cerr << "Ошибка открытия файла!" << std::endl;
            return "";
        }

        std::string line;
        while (std::getline(in, line)) {
            if (line.find("refresh_token: ") == 0) {
                in.close();
                return line.substr(15); // Удаляем "access_token: " (14 символов)
            }
        }
    }
};

struct Test {
    int Id;
    std::string Title;
    std::tm UploadDate;
    std::tm EndDate;
    int Duration; // В секундах
    int Attempts;
};

struct TestsData {
    std::vector<Test> Tests;
};

struct AnswerOption {
    int id;
    int question_id;
    std::string option_text;
    int position;
};

NLOHMANN_DEFINE_TYPE_NON_INTRUSIVE(AnswerOption, id, question_id, option_text, position)

struct Question {
    int id;
    int test_id;
    std::string question_text;
    std::string question_type;
    int points;
    int position;
    std::vector<AnswerOption> options;
};

NLOHMANN_DEFINE_TYPE_NON_INTRUSIVE(Question, id, test_id, question_text, question_type, points, position, options)

struct TestResponse {
    Test test;
    std::vector<Question> questions;
};

NLOHMANN_DEFINE_TYPE_NON_INTRUSIVE(TestResponse, test, questions)

struct UserAnswer {
    int question_id;
    nlohmann::json answer_data;
};

NLOHMANN_DEFINE_TYPE_NON_INTRUSIVE(UserAnswer, question_id, answer_data)

struct SubmitAnswer {
    std::string token;
    UserAnswer answer;
};

NLOHMANN_DEFINE_TYPE_NON_INTRUSIVE(SubmitAnswer, token, answer)

// Функция для парсинга даты из строки
bool parse_date(const std::string& date_str, std::tm& tm) {
    std::istringstream ss(date_str);
    // Формат соответствует тому, как time.Time маршалится в JSON
    ss >> std::get_time(&tm, "%Y-%m-%dT%H:%M:%SZ");
    return !ss.fail();
}

namespace nlohmann {
    template <>
    struct adl_serializer<Test> {
        static void from_json(const json& j, Test& t) {
            j.at("id").get_to(t.Id);
            j.at("title").get_to(t.Title);

            std::string upload_date_str, end_date_str;
            j.at("upload_date").get_to(upload_date_str);
            j.at("end_date").get_to(end_date_str);

            if (!parse_date(upload_date_str, t.UploadDate) ||
                !parse_date(end_date_str, t.EndDate)) {
                std::cout << "Falied to parse date" << std::endl;
            }

            j.at("duration").get_to(t.Duration);
            j.at("attempts").get_to(t.Attempts);
        }
    };

    template <>
    struct adl_serializer<TestsData> {
        static void from_json(const json& j, TestsData& td) {
            j.at("Tests").get_to(td.Tests);
        }
    };
}

void login();
void exit();
void getTestsData();
void startTest();
void sendAnswers();
bool verifyToken(const std::string &str_token);
std::string removeChar(std::string str, char ch);
bool getRadioButtonState(tgui::ScrollablePanel::Ptr panel, const tgui::String& name);
bool getCheckBoxState(tgui::ScrollablePanel::Ptr panel, const tgui::String& name);
void showDialog();

tgui::EditBox::Ptr editBoxLogin;
tgui::EditBox::Ptr editBoxPassword;

tgui::ScrollablePanel::Ptr scrollPanel;

LocalStorage localStorage;
TestsData tests_data;

TestResponse response;

int CurrentPage = 0, RealPage = 0;
// 0 - авторизация
// 1 - список тестов
// 2 - информация о тесте
// 3 - тест

bool authorized = 0, got_tests_data = 0, test_started = false, show_dialog = false, dialog_shown = false, got_questions = false;
int listViewElement = -1;

int main()
{
#if defined(_WIN32) || defined(_WIN64)
    SetConsoleCP(1251); 
    SetConsoleOutputCP(1251);
#endif

    Interface_Name = getInterfaceName();

    sf::RenderWindow window{ {1280, 720}, L"Курсовая работа Карпенко М.В." };
    tgui::Gui gui{ window };

    try {
        tgui::Font font("Inter_18pt-Regular.ttf");
        gui.setFont(font);
    }
    catch (const tgui::Exception& e) {
        std::cerr << "Failed to load font: " << e.what() << std::endl;
        //return 1;
    }

    std::string token = localStorage.GetAccessToken();

    if (verifyToken(token)) {
        CurrentPage = 1;
        RealPage = 1;

        getTestsData();

        tgui::Label::Ptr label = tgui::Label::create(u8"Доступные тесты");
        label->setPosition({ 25, 25 });
        label->setTextSize(18);

        tgui::Button::Ptr exitButton = tgui::Button::create(u8"Выйти");
        exitButton->setPosition({ 25, 650 });
        exitButton->setSize({ 60, 25 });
        exitButton->setTextSize(14);
        exitButton->onClick([]() {
            exit();
            });

        auto listView = tgui::ListView::create();
        listView->setSize(1230, 500);
        listView->setPosition(25, 60);

        // Добавление колонок
        listView->addColumn(u8"Название", 450);
        listView->addColumn(u8"Должно быть выполнено до", 200);
        listView->addColumn(u8"Ограничение по времени", 200);
        listView->addColumn(u8"Количество попыток", 150);

        // Добавление строк
        if (got_tests_data) {
            for (int i = 0; i < tests_data.Tests.size(); i++) {
                std::string duration = std::to_string(tests_data.Tests[i].Duration / 60) + u8" минут";
                listView->addItem({ tests_data.Tests[i].Title, std::asctime(&tests_data.Tests[i].EndDate), duration, std::to_string(tests_data.Tests[i].Attempts) });
            }
        }
        listView->onItemSelect([&]() {
            std::size_t selectedIndex = listView->getSelectedItemIndex();
            listViewElement = selectedIndex;
            });

        tgui::Button::Ptr startButton = tgui::Button::create(u8"Перейти к выполнению");
        startButton->setPosition({ 25, 600 });
        startButton->setSize({ 180, 25 });
        startButton->setTextSize(14);

        startButton->onClick([]() {
            std::cout << listViewElement << std::endl;
            CurrentPage = 2;
            });

        /*tgui::Button::Ptr disableButton = tgui::Button::create(u8"Выключить сетевой драйвер");
        disableButton->setPosition({ 500, 600 });
        disableButton->setSize({ 220, 25 });
        disableButton->setTextSize(14);
        disableButton->onClick(disable_network_interface);

        tgui::Button::Ptr enableButton = tgui::Button::create(u8"Включить сетевой драйвер");
        enableButton->setPosition({ 500, 650 });
        enableButton->setSize({ 220, 25 });
        enableButton->setTextSize(14);
        enableButton->onClick(enable_network_interface);*/

        gui.add(startButton, "startButton");

        /*gui.add(disableButton);
        gui.add(enableButton);*/

        gui.add(listView, "tests");
        gui.add(label);
        
        gui.add(exitButton, "exitButton");
    }
    else {
        CurrentPage = 0;
        RealPage = 0;
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

        /*tgui::Button::Ptr disable = tgui::Button::create(u8"Отключить сетевой драйвер");
        disable->setPosition({ 25, 25 });
        disable->setSize({ 210, 25 });
        disable->setTextSize(14);
        disable->onClick(disable_network_interface);

        tgui::Button::Ptr enable = tgui::Button::create(u8"Включить сетевой драйвер");
        enable->setPosition({ 25, 75 });
        enable->setSize({ 210, 25 });
        enable->setTextSize(14);
        enable->onClick(enable_network_interface);

        gui.add(disable);
        gui.add(enable);*/

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

            if (event.type == sf::Event::Resized) {
                gui.get("startButton")->setPosition({ 25, window.getSize().y - 120 });
                gui.get("exitButton")->setPosition({25, window.getSize().y - 70});
                gui.get("tests")->setSize({window.getSize().x - 50, window.getSize().y - 220 });
            }

            if (event.type == sf::Event::Closed)
                window.close();
        }

        // Модальное окно
        tgui::ChildWindow::Ptr dialog = tgui::ChildWindow::create();
        dialog->setTitle(u8"Предупреждение");
        dialog->setSize(400, 150);
        dialog->setPosition("50%", "50%");
        dialog->setOrigin(0.5f, 0.5f);
        dialog->setPositionLocked(true);
        dialog->setResizable(false);

        // Поле ввода
        auto labelAttention = tgui::Label::create(u8"Вы действительно хотите завершить тестирование?");
        labelAttention->setPosition("10%", "20%");
        dialog->add(labelAttention);

        // Кнопки
        auto btnOk = tgui::Button::create(u8"Да");
        btnOk->setSize(80, 30);
        btnOk->setPosition("50% - 90", "70%");
        btnOk->onPress([dialog] {
            sendAnswers();
            dialog->close();
            });
        dialog->add(btnOk);

        auto btnCancel = tgui::Button::create(u8"Отмена");
        btnCancel->setSize(80, 30);
        btnCancel->setPosition("50% + 10", "70%");
        btnCancel->onPress([dialog] { show_dialog = false; dialog_shown = false; dialog->close(); });
        dialog->add(btnCancel);

        if (CurrentPage == 0 && RealPage != 0) {
            RealPage = 0;
            gui.removeAllWidgets();

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
            CurrentPage = 0;
        }

        if (CurrentPage == 1 && RealPage != 1) {
            RealPage = 1;
            getTestsData();
            gui.removeAllWidgets();

            tgui::Label::Ptr label = tgui::Label::create(u8"Доступные тесты");
            label->setPosition({ 25, 25 });
            label->setTextSize(18);

            tgui::Button::Ptr exitButton = tgui::Button::create(u8"Выйти");
            exitButton->setPosition({ 25, 650 });
            exitButton->setSize({ 60, 25 });
            exitButton->setTextSize(14);
            exitButton->onClick([]() {
                exit();
                });

            auto listView = tgui::ListView::create();
            listView->setSize(1230, 500);
            listView->setPosition(25, 60);

            // Добавление колонок
            listView->addColumn(u8"Название", 450);
            listView->addColumn(u8"Должно быть выполнено до", 200);
            listView->addColumn(u8"Ограничение по времени", 200);
            listView->addColumn(u8"Количество попыток", 150);

            // Добавление строк
            if (got_tests_data) {
                for (int i = 0; i < tests_data.Tests.size(); i++) {
                    std::string duration = std::to_string(tests_data.Tests[i].Duration / 60) + u8" минут";
                    listView->addItem({ tests_data.Tests[i].Title, std::asctime(&tests_data.Tests[i].EndDate), duration, std::to_string(tests_data.Tests[i].Attempts) });
                }
            }

            listView->onItemSelect([&]() {
                std::size_t selectedIndex = listView->getSelectedItemIndex();
                listViewElement = selectedIndex;
                });

            tgui::Button::Ptr startButton = tgui::Button::create(u8"Перейти к выполнению");
            startButton->setPosition({ 25, 600 });
            startButton->setSize({ 180, 25 });
            startButton->setTextSize(14);
            startButton->onClick([]() {
                std::cout << listViewElement << std::endl;
                CurrentPage = 2;
                });

            gui.add(startButton, "startButton");

            gui.add(listView, "tests");

            gui.add(label);
            gui.add(exitButton, "exitButton");
        }

        if (CurrentPage == 2 && RealPage != 2) {
            RealPage = 2;

            gui.removeAllWidgets();
            tgui::Label::Ptr label = tgui::Label::create(tests_data.Tests[listViewElement].Title);
            label->setPosition({ 25, 25 });
            label->setTextSize(24);

            std::string duration = std::to_string(tests_data.Tests[listViewElement].Duration / 60) + u8" минут";
            tgui::Label::Ptr info = tgui::Label::create(u8"Попыток: " + std::to_string(tests_data.Tests[listViewElement].Attempts) + u8" Ограничение по времени: " + duration);
            info->setPosition({ 25, 70 });
            info->setTextSize(14);

            tgui::Button::Ptr startButton = tgui::Button::create(u8"Перейти к выполнению");
            startButton->setPosition({ 25, 100 });
            startButton->setSize({ 180, 25 });
            startButton->setTextSize(14);
            startButton->onClick(startTest);

            tgui::Button::Ptr backButton = tgui::Button::create(u8"Вернуться");
            backButton->setPosition({ 25, 150 });
            backButton->setSize({ 100, 25 });
            backButton->setTextSize(14);
            backButton->onClick([]() {
                CurrentPage = 1;
                });

            gui.add(label);
            gui.add(info);
            gui.add(startButton);
            gui.add(backButton);
        }

        if (CurrentPage == 3 && RealPage != 3) {
            RealPage = 3;

            gui.removeAllWidgets();
            tgui::Label::Ptr label = tgui::Label::create(response.test.Title); // Заголовок теста
            label->setPosition({ 25, 25 });
            label->setTextSize(24);

            scrollPanel = tgui::ScrollablePanel::create();
            scrollPanel->setSize(1230, 600); // Размер области
            scrollPanel->setPosition(25, 70); // Позиция на экране
            scrollPanel->getRenderer()->setBackgroundColor(tgui::Color(240, 240, 240)); // Фон

            gui.add(scrollPanel);

            int offset = 0;
            for (int i = 0; i < response.questions.size(); i++) {
                std::string additive_text;
                if (response.questions[i].question_type == "single_choice")
                    additive_text = u8" (Выберите один правильный ответ)";
                else if (response.questions[i].question_type == "multiple_choice")
                    additive_text = u8" (Выберите один или несколько правильных ответов)";
                tgui::Label::Ptr questionLabel = tgui::Label::create(response.questions[i].question_text + additive_text); // Заголовок вопроса

                questionLabel->setPosition({ 25, 30 + offset });
                questionLabel->setTextSize(14);

                scrollPanel->add(questionLabel);

                offset += 30; // Добавляем смещение

                if (response.questions[i].question_type == "single_choice") {
                    tgui::RadioButtonGroup radioGroup;

                    for (int j = 0; j < response.questions[i].options.size(); j++) {
                        tgui::RadioButton::Ptr choice = tgui::RadioButton::create(); // Выбор одного варианта

                        choice->setPosition({ 30, 30 + offset });

                        tgui::Label::Ptr choiceLabel = tgui::Label::create(response.questions[i].options[j].option_text);
                        choiceLabel->setPosition({ 50, 30 + offset });
                        choiceLabel->setTextSize(14);

                        offset += 20;

                        scrollPanel->add(choice, std::to_string(response.questions[i].id) + "/" + std::to_string(response.questions[i].options[j].id));
                        scrollPanel->add(choiceLabel);
                    }
                }
                else if (response.questions[i].question_type == "multiple_choice") {
                    for (int j = 0; j < response.questions[i].options.size(); j++) {
                        tgui::CheckBox::Ptr choice = tgui::CheckBox::create(response.questions[i].options[j].option_text); // Выбор вариантов

                        choice->setPosition({ 30, 30 + offset });

                        offset += 20;

                        scrollPanel->add(choice, std::to_string(response.questions[i].id) + "/" + std::to_string(response.questions[i].options[j].id));
                    }
                }
                else if (response.questions[i].question_type == "text_answer") {
                    tgui::EditBox::Ptr textArea = tgui::EditBox::create();

                    textArea->setPosition({ 30, 30 + offset });
                    textArea->setSize({ 160, 20 });
                    textArea->setTextSize(14);

                    offset += 30;

                    scrollPanel->add(textArea, std::to_string(response.questions[i].id));
                }

                offset += 30;
            }

            gui.add(label);

            // Кнопка Завершить тестирование

            tgui::Button::Ptr endButton = tgui::Button::create(u8"Завершить тестирование");
            endButton->setPosition({ 25, 680 });
            endButton->setSize({ 200, 25 });

            

            endButton->onClick([dialog] {
                showDialog();
                });
            

            gui.add(endButton);
        }

        if (show_dialog && !dialog_shown) {
            gui.add(dialog);
            dialog_shown = true;
        }

        window.clear(sf::Color(255, 255, 255, 255));

        gui.draw();

        window.display();
    }
}

void showDialog() {
    show_dialog = true;
}

bool getRadioButtonState(tgui::ScrollablePanel::Ptr panel, const std::string& name)
{
    // Получаем виджет по имени из панели
    tgui::Widget::Ptr widget = panel->get(name);

    if (!widget)
        throw std::runtime_error("Widget '" + name + "' not found!");

    // Пробуем преобразовать к RadioButton
    auto radioButton = widget->cast<tgui::RadioButton>();

    if (!radioButton)
        throw std::runtime_error("Widget '" + name + "' is not a RadioButton!");

    return radioButton->isChecked();
}

bool getCheckBoxState(tgui::ScrollablePanel::Ptr panel, const std::string& name)
{
    // Получаем виджет по имени из панели
    tgui::Widget::Ptr widget = panel->get(name);

    if (!widget)
        throw std::runtime_error("Widget '" + name + "' not found!");

    // Пробуем преобразовать к RadioButton
    auto checkBox = widget->cast<tgui::CheckBox>();

    if (!checkBox)
        throw std::runtime_error("Widget '" + name + "' is not a RadioButton!");

    return checkBox->isChecked();
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

    // POST-запрос с JSON
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
    str.erase(std::remove(str.begin(), str.end(), ch), str.end());
    return str;
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
        // Парсим и сохраняем токены
        nlohmann::json j = nlohmann::json::parse(post_res->body);

        std::string access_token = j["access_token"].get<std::string>();
        std::string refresh_token = j["refresh_token"].get<std::string>();
        
        localStorage.SetTokens(access_token, refresh_token);
        authorized = true;
        CurrentPage = 1;
        std::cout << "Авторизован. Статус код 200" << std::endl;
    }
    else {
        std::cout << "Invalid credentials" << std::endl;
    }
}

void exit() {
    localStorage.Clear();
    authorized = false;
    CurrentPage = 0;
}

void getTestsData() {
    if (!authorized) {
        return;
    }

    httplib::Client cli("localhost:8080");

    // POST-запрос с JSON
    httplib::Headers headers = {
        {"Authorization", localStorage.GetAccessToken()},
        {"Content-Type", "application/json"}
    };
    
    auto post_res = cli.Post("/api/gettestsdata", headers, "", "application/json");

    if (post_res && post_res->status == 200) {
        //std::cout << post_res->body << std::endl;
        try {
            // Парсинг JSON
            nlohmann::json j = nlohmann::json::parse(post_res->body);
            tests_data = j.get<TestsData>();

            // Вывод данных
            for (const auto& test : tests_data.Tests) {
                std::cout << "id: " << test.Id << "\n" 
                    << "Title: " << test.Title << "\n"
                    << "Upload Date: " << std::asctime(&test.UploadDate)
                    << "End Date: " << std::asctime(&test.EndDate)
                    << "Duration: " << test.Duration << "\n"
                    << "Attempts: " << test.Attempts << "\n\n";
            }

            got_tests_data = true;
        }
        catch (const nlohmann::json::exception& e) {
            std::cerr << "JSON error: " << e.what() << std::endl;
        }
        catch (const std::exception& e) {
            std::cerr << "Error: " << e.what() << std::endl;
        }
    }
    else {
        std::cout << "Failed to fetch tests data." << std::endl;
    }
}

void getTestData(int selected_listview_item) {
    int id = tests_data.Tests[selected_listview_item].Id;

    httplib::Client cli("localhost:8080");

    // Выполнение GET-запроса
    auto res = cli.Get("/api/tests/test/" + std::to_string(id));

    if (res && res->status == 200) {
        try {
            // Парсинг JSON ответа
            nlohmann::json j = nlohmann::json::parse(res->body);

            // Конвертация JSON в структуры данных
            response = j.get<TestResponse>();

            // Пример работы с данными
            std::cout << "Test title: " << response.test.Title << std::endl;
            std::cout << "Questions count: " << response.questions.size() << std::endl;

            // Сохранение в файл
            std::ofstream output("test_data.json");
            output << std::setw(4) << j << std::endl;
            output.close();

            std::cout << "Data saved to test_data.json" << std::endl;
            got_questions = true;

        }
        catch (const std::exception& e) {
            std::cout << res->body << std::endl;
            std::cerr << "JSON parse error: " << e.what() << std::endl;
        }
    }
    else {
        std::cerr << "Request failed with status: "
            << res->status << std::endl;
        got_questions = false;
    }
}

int findQuestionId(const std::string& element_name) {
    size_t slash_pos = element_name.find('/');
    if (slash_pos == std::string::npos) {
        return 0;
    }
    return std::stoi(element_name.substr(0, slash_pos));
}

int findChoiceId(const std::string& element_name) {
    size_t slash_pos = element_name.find('/');
    if (slash_pos == std::string::npos) {
        return 0;
    }
    return std::stoi(element_name.substr(slash_pos + 1));
}

void startTest() {
    getTestData(listViewElement);
    
    if (got_questions) {
        test_started = true;
        CurrentPage = 3;

        // Начинаем попытку
        httplib::Client cli("localhost:8080");

        // POST-запрос с JSON
        httplib::Headers headers = {
            {"Content-Type", "application/json"}
        };
        std::string body = R"({"id": ")" + std::to_string(listViewElement) + R"(", "token": ")" + localStorage.GetAccessToken() + R"("})";

        auto post_res = cli.Post("/api/tests/startattempt", headers, body, "application/json");
        if (post_res && post_res->status == 200) {
            test_started = true;
            CurrentPage = 3;
        }
        else {
            test_started = false;
        }
    }
}

void sendAnswers() {
    // Отправление ответов
    httplib::Client cli("localhost:8080");

    // POST-запрос с JSON
    httplib::Headers headers = {
        {"Content-Type", "application/json"}
    };

    for (int i = 0; i < response.questions.size(); i++) {
        nlohmann::json answer_json;
        if (response.questions[i].question_type == "single_choice") {
            int opt_id = 0;
            for (int j = 0; j < response.questions[i].options.size(); j++) {
                if (getRadioButtonState(scrollPanel, std::to_string(response.questions[i].id) + "/" + std::to_string(response.questions[i].options[j].id))) {
                    opt_id = response.questions[i].options[j].id;
                }
            }
            answer_json["selected_option_id"] = opt_id;
        }
        else if (response.questions[i].question_type == "multiple_choice") {
            std::vector<int> opt_ids;
            for (int j = 0; j < response.questions[i].options.size(); j++) {
                if (getCheckBoxState(scrollPanel, std::to_string(response.questions[i].id) + "/" + std::to_string(response.questions[i].options[j].id))) {
                    opt_ids.push_back(response.questions[i].options[j].id);
                }
            }
            answer_json["selected_option_ids"] = opt_ids;
        }
        else if (response.questions[i].question_type == "text_answer") {
            std::string text;

            tgui::Widget::Ptr widget = scrollPanel->get(std::to_string(response.questions[i].id));

            auto textfield = widget->cast<tgui::EditBox>();

            if (textfield)
                text = textfield->getText().toStdString();
            answer_json["answer_text"] = text;
        }
        UserAnswer answer{ response.questions[i].id, answer_json };
        SubmitAnswer data{ localStorage.GetAccessToken(), answer };
        nlohmann::json j = data;

        std::cout << j.dump() << std::endl;

        auto post_res = cli.Post("/api/attempts/answer", headers, j.dump(), "application/json");
        if (post_res && post_res->status == 200) {
            std::cout << "Ответ успешно отправлен." << std::endl;
        }
        else {
            std::cout << "Ошибка при отправке ответа." << std::endl;
        }
    }

    nlohmann::json j;
    j["token"] = localStorage.GetAccessToken();
    j["test_id"] = tests_data.Tests[listViewElement].Id;

    auto post_res = cli.Post("/api/attempts/finish", headers, j.dump(), "application/json");
    if (post_res && post_res->status == 200) {
        std::cout << "Попытка успешно завершена." << std::endl;
    }
    else {
        std::cout << "Ошибка при завершении попытки." << std::endl;
    }

    CurrentPage = 1;
}