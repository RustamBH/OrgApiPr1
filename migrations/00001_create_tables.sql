-- +goose Up
-- Таблица подразделений
CREATE TABLE IF NOT EXISTS departments (
	id SERIAL PRIMARY KEY,
	name VARCHAR(200) NOT NULL,
	parent_id INTEGER REFERENCES departments(id) ON DELETE CASCADE,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Уникальность имен в рамках одного родителя
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_parent_name 
ON departments(parent_id, LOWER(TRIM(name)));

-- Таблица сотрудников
CREATE TABLE IF NOT EXISTS employees (
	id SERIAL PRIMARY KEY,
	department_id INTEGER NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
	full_name VARCHAR(200) NOT NULL,
	position VARCHAR(200) NOT NULL,
	hired_at DATE,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индекс для быстрого поиска сотрудников по отделу
CREATE INDEX IF NOT EXISTS idx_employees_department_id ON employees(department_id);

-- +goose Down
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS departments;
