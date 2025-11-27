package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataBaseConnection(t *testing.T) {
	initTestDB()
	defer db.Close()

	var result int
	err := db.QueryRow("SELECT 1").Scan(&result)
	assert.NoError(t, err, "Дорлжны уметь выполнять SQL запросы")
	assert.Equal(t, 1, result, "Должны получить 1")
}

func TestTableExists(t *testing.T) {
	setupTest(t)

	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name = 'users'
		)
	`).Scan(&tableExists)

	assert.NoError(t, err, "Должны проверить существование таблицы")
	assert.True(t, tableExists, "Таблица users должна существовать")
}
