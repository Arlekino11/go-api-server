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
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDB *sql.DB

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
	setupTest(t)

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
	err = json.Unmarshal(rr.Body.Bytes(), &createdUser)
	assert.NoError(t, err, "Должен корректно распарсить JSON ответ")

	assert.Equal(t, user.Name, createdUser.Name, "Имя должно совпадать")
	assert.Equal(t, user.Email, createdUser.Email, "Email должен совпадать")
	assert.NotZero(t, createdUser.ID, "ID Должен быть установлен")

	//read test

	reqGetOne, err := http.NewRequest("GET", "/users/"+strconv.Itoa(createdUser.ID), nil)
	assert.NoError(t, err)

	rrGetOne := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/users/{id}", getUserHandler).Methods("GET")
	router.ServeHTTP(rrGetOne, reqGetOne)

	assert.Equal(t, http.StatusOK, rrGetOne.Code, "Должен вернуть статус 200")

	var fetchedUser User
	err = json.Unmarshal(rrGetOne.Body.Bytes(), &fetchedUser)
	assert.NoError(t, err, "Должен корректно распарсить JSON ответ")
	assert.Equal(t, createdUser.ID, fetchedUser.ID)
	assert.Equal(t, createdUser.Name, fetchedUser.Name)
	assert.Equal(t, createdUser.Email, fetchedUser.Email)
}

func TestMain(m *testing.M) {
	os.Setenv("DB_NAME", "go_learning_test")

	fmt.Println("Инициализация тестовой базы данных...")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", "localhost", 5433, "postgres", "admin", "go_learning_test")

	var err error
	testDB, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Printf("Не удалось подключиться к тестовой базе: %v\n", err)
		os.Exit(1)
	}

	err = testDB.Ping()
	if err != nil {
		fmt.Printf("Не удалось проверить подключение к тестовой базе: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Тестовая база данных инициализирована")

	code := m.Run()

	testDB.Close()
	fmt.Printf("Тестовая база данных закрыта")
	os.Exit(code)
}

func setupTest(t *testing.T) {
	db = testDB

	_, err := db.Exec("DELETE FROM users")
	require.NoError(t, err, "Должны очистить таблицу перед тестом")

	_, err = db.Exec("ALTER SEQUENCE users_id_seq RESTART WITH 1")
	require.NoError(t, err, "Должны сбросить sequence")
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

func TestCreateUser_InvalidJSON(t *testing.T) {
	setupTest(t)

	invalidJSON := `{"name": "Test", "email": "test@example.com"`

	req, err := http.NewRequest("POST", "/users", bytes.NewBufferString(invalidJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)

	var errorResp ErrorResponse
	json.Unmarshal(rr.Body.Bytes(), &errorResp)
	assert.Equal(t, CodeValidationError, errorResp.Code)
	assert.Contains(t, errorResp.Message, "Invalid JSON")
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	setupTest(t)

	user1 := User{Name: "User1", Email: "duplicate@example.com"}
	createUserInDB(t, user1)

	user2 := User{Name: "User2", Email: "duplicate@example.com"}
	jsonData, _ := json.Marshal(user2)

	req, err := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content_Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)

	var errorResp ErrorResponse
	json.Unmarshal(rr.Body.Bytes(), &errorResp)
	assert.Equal(t, CodeDuplicateEntry, errorResp.Code)
	assert.Contains(t, errorResp.Message, "already exist")
}

func TestGetUser_NotFound(t *testing.T) {
	setupTest(t)

	req, err := http.NewRequest("GET", "/users/99999", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/users/{id}", getUserHandler).Methods("GET")
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code, "Должен вернуть 404 для несуществующего пользователя")

	var errorResp ErrorResponse
	json.Unmarshal(rr.Body.Bytes(), &errorResp)
	assert.Equal(t, CodeNotFound, errorResp.Code)
	assert.Contains(t, errorResp.Message, "not found")
}

func createUserInDB(t *testing.T, user User) {
	sqlStatement := `INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id`

	err := db.QueryRow(sqlStatement, user.Name, user.Email).Scan(&user.ID)
	assert.NoError(t, err)
}
