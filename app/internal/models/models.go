package models

import (
	"html/template"
	"time"
)

// Данные для входа
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserData struct {
	Username  string `json:"Username"`
	Password  string `json:"Password"`
	Role      string `json:"Role"`
	GroupName string `json:"GroupName"`
}

type GroupData struct {
	GroupName string `json:"GroupName"`
}

type TokenResponse struct {
	AccessToken string `json:"Authorization"`
}

type ProfilePageData struct {
	Username string `json:"Username"`
	Group    string `json:"Group"`
}

type TeacherCoursesPageData struct {
	Courses []string `json:"Courses"`
}

type Course struct {
	Name  string `json:"Name"`
	Files []File `json:"Files"`
	Tests []Test `json:"Tests"`
}

type File struct {
	Name       string    `json:"Name"`
	UploadDate time.Time `json:"UploadDate"`
}

type CoursesPageData struct {
	Courses []string `json:"Courses"`
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

// Данные на админ панели(группы и пользователи)
type AdminPanelData struct {
	Groups []Group `json:"Groups"`
	Users  []User  `json:"Users"`
}

type ServeAdminPanelData struct {
	Groups                  template.HTML `json:"Groups"`
	GroupsTable             template.HTML `json:"GroupsTable"`
	UsersTable              template.HTML `json:"UsersTable"`
	ВсегоПосещений          int64         `json:"ВсегоПосещений"`
	СамаяПопулярнаяСтраница string        `json:"СамаяПопулярнаяСтраница"`
}

type Statistics struct {
	ПосещенияПрофль      int64 `json:"ПосещенияПрофль"`
	ПосещенияАдминПанель int64 `json:"ПосещенияАдминПанель"`
	ПосещенияОценки      int64 `json:"ПосещенияОценки"`
	ПосещенияКурсы       int64 `json:"ПосещенияКурсы"`
}

type StatsToView struct {
	ВсегоПосещений          int64  `json:"ВсегоПосещений"`
	СамаяПопулярнаяСтраница string `json:"СамаяПопулярнаяСтраница"`
}

var Stats Statistics

type CoursesPageServeData struct {
	Courses template.HTML `json:"courses"`
}

type DeleteGroupData struct {
	Id string `json:"Id"`
}

type DeleteUserData struct {
	Token string `json:"token"`
	Name  string `json:"Username"`
}

type DeleteUser struct {
	Name string `json:"Username"`
}

// Тесты
type Test struct {
	Id         int       `json:"Id"`
	Title      string    `json:"Title"`
	UploadDate time.Time `json:"UploadDate"`
	EndDate    time.Time `json:"EndDate"`
	Duration   string    `json:"Duration"`
	Attempts   int       `json:"Attempts"`
}

type TestsData struct {
	Tests []Test `json:"Tests"`
}
