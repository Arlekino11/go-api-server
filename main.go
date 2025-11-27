package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var dbConfig = struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}{
	Host:     getEnv("DB_HOST", "localhost"),
	Port:     getEnvInt("DB_PORT", 5433),
	User:     getEnv("DB_USER", "postgres"),
	Password: getEnv("DB_PASSWORD", "admin"),
	DBName:   getEnv("DB_NAME", "go_learning"),
}

func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

const (
	host     = "localhost"
	port     = 5433
	user     = "postgres"
	password = "admin"
	dbname   = "go_learning"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var db *sql.DB

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Ошибка подключения:", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Не удалось подключится к базе:", err)
	}
	fmt.Println("Успешное подключение к PostgreSQL!")

	router := mux.NewRouter()

	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/users", getUsersHandler).Methods("GET")
	router.HandleFunc("/users", createUserHandler).Methods("POST")
	router.HandleFunc("/users/{id}", getUserHandler).Methods("GET")
	router.HandleFunc("/users/{id}", updateUserHandler).Methods("PUT")
	router.HandleFunc("/users/{id}", deleteUserHandler).Methods("DELETE")

	fmt.Println("Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Получен запрос на /users")

	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		log.Printf("Ошибка запроса: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Name, &user.Email)
		if err != nil {
			log.Printf("Ошибка сканирования: %v", err)
			continue
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Ошибка итерации: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
	fmt.Printf("Отправлено %d пользователей\n", len(users))
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Получен POST запрос на /users")

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	if user.Name == "" || user.Email == "" {
		http.Error(w, "Имя и Email обязательны", http.StatusBadRequest)
		return
	}

	sqlStatement := `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id`

	id := 0
	err = db.QueryRow(sqlStatement, user.Name, user.Email).Scan(&id)
	if err != nil {
		log.Printf("Ошибка вставки: %v", err)
		http.Error(w, "Ошибка при создании пользователя", http.StatusInternalServerError)
		return
	}

	user.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
	fmt.Printf("Создан пользователь: %s {ID: %d}\n", user.Name, user.ID)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	fmt.Printf("Получен GET запрос на /users/%s\n", idStr)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный id дользователя", http.StatusBadRequest)
		return
	}

	var user User
	err = db.QueryRow("SELECT id, name, email FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
		} else {
			log.Printf("Ошибка запроса: %v", err)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	fmt.Printf("Отправлен пользователь; %s\n", user.Name)
}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	fmt.Printf("Получен PUT запрос на /users/%s\n", idStr)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	var user User
	err = json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Неверный JSON", http.StatusBadRequest)
		return
	}

	if user.Name == "" || user.Email == "" {
		http.Error(w, "Имя и email обязательны", http.StatusBadRequest)
		return
	}

	sqlStatement := `
        UPDATE users 
        SET name = $1, email = $2 
        WHERE id = $3
        RETURNING id, name, email`

	updatedUser := User{}
	err = db.QueryRow(sqlStatement, user.Name, user.Email, id).Scan(&updatedUser.ID, &updatedUser.Name, &updatedUser.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Пользователь не найден", http.StatusNotFound)
		} else {
			log.Printf("Ошибка обновления: %v", err)
			http.Error(w, "Ошибка при обновлении пользователя", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
	fmt.Printf("Обновлен пользователь: %s (ID: %d)\n", updatedUser.Name, updatedUser.ID)
}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	fmt.Printf("Получен DELETE запрос на /user/%s\n", idStr)

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID пользователя", http.StatusBadRequest)
		return
	}

	sqlStatement := "DELETE FROM users WHERE id = $1"
	result, err := db.Exec(sqlStatement, id)
	if err != nil {
		log.Printf("Ошибка удаления: %v", err)
		http.Error(w, "Ошибка при удалении пользователя", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Ошибка получения количесва удаленных строк: %v", err)
		http.Error(w, "Ошибка удаления пользователя", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
	fmt.Printf("Удален пользователь с ID: %d\n", id)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Добро пожаловать в Users API! Доступные эндпоинты:\n\nGET /users - список пользователей\nPOST /users - создать пользователя\nGET /users/{id} - получить пользователя\nPUT /users/{id} - обновить пользователя\nDELETE /users/{id} - удалить пользователя")
}
