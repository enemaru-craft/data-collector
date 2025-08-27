package model

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func InitDB() (*sql.DB, error) {
	var conn *sql.DB
	/* TODO トラフィックが少ないのでこの方法でできるが､多くなる場合はコネクションについて再考する必要がある*/
	// すでにコネクションが存在している場合は再利用する
	user := "postgres"
	pass := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	port := "5432"
	name := "stg"

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		host, port, user, pass, name)

	var err error
	conn, err = sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open DB: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}

	return conn, nil
}
