package db

import (
	"database/sql"
	"fmt"
)

// SeedSampleData populates the database with a standard Office schema for testing.
func SeedSampleData(conn *sql.DB, cfg *Config) error {
	var sqlScript string

	switch cfg.Dialect {
	case "postgres":
		sqlScript = `
			DROP TABLE IF EXISTS employees CASCADE;
			DROP TABLE IF EXISTS departments CASCADE;

			CREATE TABLE departments (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL
			);

			CREATE TABLE employees (
				id SERIAL PRIMARY KEY,
				name TEXT NOT NULL,
				salary NUMERIC,
				dept_id INTEGER REFERENCES departments(id)
			);

			INSERT INTO departments (name) VALUES ('Engineering'), ('Marketing'), ('Sales');
			INSERT INTO employees (name, salary, dept_id) VALUES 
				('Alice', 90000, 1),
				('Bob', 80000, 1),
				('Charlie', 70000, 2),
				('David', 65000, 3);
		`
	default:
		return fmt.Errorf("seeding not supported for dialect: %s", cfg.Dialect)
	}

	_, err := conn.Exec(sqlScript)
	return err
}
