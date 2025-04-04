package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	run()
}

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Message string `json:"message"`
}

// Обработчик для api/login
func handleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginData LoginData
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		sendError(w, "Invalid request", http.StatusBadRequest)
		return
	}
	//fmt.Printf("login: " + loginData.Username + " password: " + loginData.Password)
	// Простая логика проверки (в реальности используйте БД и хеширование)
	if loginData.Username == "admin" && loginData.Password == "12345" {
		sendResponse(w, Response{Message: "Success"}, http.StatusOK)
		fmt.Println("ADMIN SIGNED IN")
	} else {
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

func sendResponse(w http.ResponseWriter, resp Response, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Message: message})
}

func run() {
	// Регистрируем обработчики для маршрутов
	http.HandleFunc("/api/auth", handleAuth)
	//http.HandleFunc("/about", aboutHandler)

	// Указываем адрес и порт для прослушивания
	addr := ":1337" // Можно изменить порт на другой свободный

	// Выводим сообщение в консоль
	fmt.Printf("Сервер запущен и слушает на %s\n", addr)

	// Запускаем сервер
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("Ошибка запуска сервера: ", err)
	}
}
