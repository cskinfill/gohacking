package main

import (
	"context"
	"database/sql"
	"log"
)

type DbRepo struct {
	db *sql.DB
}

func NewDbRepo(db *sql.DB) (*DbRepo, error) {
	// db, err := otelsql.Open("sqlite3", database)
	return &DbRepo{
		db: db,
	}, nil
}

func (r *DbRepo) Services(ctx context.Context) ([]Service, error) {
	_, span := tracer.Start(ctx, "Services")
	defer span.End()
	rows, err := r.db.QueryContext(ctx, "SELECT * FROM services")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := []Service{}
	for rows.Next() {
		service := Service{}
		err = rows.Scan(&service.ID, &service.Name, &service.Description, &service.Versions)
		if err != nil {
			return nil, err
		}
		data = append(data, service)
	}
	return data, nil
}

func (r *DbRepo) Service(ctx context.Context, id int) (*Service, error) {
	_, span := tracer.Start(ctx, "Service")
	defer span.End()
	row := r.db.QueryRowContext(ctx, "SELECT * FROM services WHERE id=?", id)

	// Parse row into Activity struct
	service := Service{}
	var err error
	if err = row.Scan(&service.ID, &service.Name, &service.Description, &service.Versions); err == sql.ErrNoRows {
		log.Printf("Id not found")
		return nil, nil
	}

	if err != nil {
		log.Printf("Something bad - %s", err)
		return nil, err
	}
	return &service, nil
}
