package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomeHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(homeHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Добро пожаловать в Users API")
}

func TestUserCRUD(t *testing.T) {
	if db == nil {
		initTestDB()
	}

	_, err := db.Exec("DELETE FROM users")
	assert.NoError(t, err, "Должны очистить таблицу перед тестом")

	user := User{
		Name:  "Test User",
		Email: "test@example.com",
	}
	jsonData, _ := json.Marshal(user)

	//create test

	req, err := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code, "Должен вернуть статус 201")

	var createdUser User
	json.Unmarshal(rr.Body.Bytes(), &createdUser)
	assert.NoError(t, err, "Должен корректно распарсить JSON ответ")

	assert.Equal(t, user.Name, createdUser.Name, "Имя должно совпадать")
	assert.Equal(t, user.Email, createdUser.Email, "Email должен совпадать")
	assert.NotZero(t, createdUser.ID, "ID Должен быть установлен")

	//read test

	reqGet, err := http.NewRequest("GET", "/users", nil)
	assert.NoError(t, err)

	rrGet := httptest.NewRecorder()
	handlerGet := http.HandlerFunc(getUsersHandler)
	handlerGet.ServeHTTP(rrGet, reqGet)

	assert.Equal(t, http.StatusOK, rrGet.Code)

	var users []User
	json.Unmarshal(rrGet.Body.Bytes(), &users)
	assert.NoError(t, err)

	assert.Greater(t, len(users), 0, "Должен вернуть хотя бы одного пользователя")
	assert.Equal(t, createdUser.ID, users[len(users)-1].ID)
}

func TestMain(m *testing.M) {
	os.Setenv("DB_NAME", "go_learning_test")

	initTestDB()
	defer db.Close()

	code := m.Run()
	os.Exit(code)
}

func initTestDB() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "localhost", 5433, "postgres", "admin", "go_learning_test")

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Не удалось проверить подключение к тестовой базе: %v", err)
	}
	fmt.Println("Тестовая база данных инициализирована")
}
