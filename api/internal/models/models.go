package models

import (
	"time"
)

// Данные для входа
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type GroupData struct {
	GroupName string `json:"GroupName"`
}

type DeleteGroupData struct {
	Id string `json:"Id"`
}

// Ответ сервера
type Response struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// Токен
type TokenResponse struct {
	AccessToken string `json:"Authorization"`
}

// Оценка
type Mark struct {
	Course    string    `json:"Course"`
	Date      time.Time `json:"Date"`
	MarkValue string    `json:"MarkValue"`
}

// Данные страницы профиля
type ProfilePageData struct {
	Username string   `json:"Username"`
	Group    string   `json:"Group"`
	Courses  []Course `json:"Courses"`
}

type Course struct {
	Id    int    `json:"id"`
	Name  string `json:"Name"`
	Files []File `json:"Files"`
	Tests []Test `json:"Tests"`
}

type File struct {
	Id         int       `json:"Id"`
	Name       string    `json:"Name"`
	UploadDate time.Time `json:"UploadDate"`
	Filename   string    `json:"Filename"`
}

type ServeCoursePage struct {
	Course Course `json:"course"`
}

type TeacherCoursesPageData struct {
	Courses []Course `json:"Courses"`
	Groups  []Group  `json:"Groups"`
}

type CoursesPageData struct {
	Courses []Course `json:"Courses"`
}

// Группа
type Group struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

// Пользователь
type User struct {
	Id        int    `json:"id"`
	Username  string `json:"Username"`
	Role      string `json:"Role"`
	GroupName string `json:"GroupName"`
}

type UserData struct {
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	Role      string `json:"Role"`
	GroupName string `json:"GroupName"`
}

// Данные на админ панели(группы и пользователи)
type AdminPanelData struct {
	Groups []Group `json:"Groups"`
	Users  []User  `json:"Users"`
}

type DeleteUserData struct {
	Name string `json:"Username"`
}

type TestsData struct {
	Tests []Test `json:"Tests"`
}
