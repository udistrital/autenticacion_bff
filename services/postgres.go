package services

import (
	"database/sql"
	"fmt"
	"sync"

	beego "github.com/beego/beego/v2/server/web"
	_ "github.com/lib/pq"
)

var (
	postgresDB   *sql.DB
	postgresOnce sync.Once
	postgresErr  error
)

func GetPostgresDB() (*sql.DB, error) {
	postgresOnce.Do(func() {
		host := beego.AppConfig.DefaultString("postgres_host", "")
		port := beego.AppConfig.DefaultString("postgres_port", "5432")
		dbname := beego.AppConfig.DefaultString("postgres_db", "")
		user := beego.AppConfig.DefaultString("postgres_user", "")
		password := beego.AppConfig.DefaultString("postgres_password", "")
		sslmode := beego.AppConfig.DefaultString("postgres_sslmode", "disable")

		if host == "" || dbname == "" || user == "" || password == "" {
			postgresErr = fmt.Errorf("la configuración de PostgreSQL está incompleta")
			return
		}

		dsn := fmt.Sprintf(
			"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
			host,
			port,
			dbname,
			user,
			password,
			sslmode,
		)

		db, err := sql.Open("postgres", dsn)
		if err != nil {
			postgresErr = fmt.Errorf("error abriendo conexión PostgreSQL: %w", err)
			return
		}

		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(5)

		if err := db.Ping(); err != nil {
			_ = db.Close()
			postgresErr = fmt.Errorf("error validando conexión PostgreSQL: %w", err)
			return
		}

		postgresDB = db
	})

	if postgresErr != nil {
		return nil, postgresErr
	}
	return postgresDB, nil
}
