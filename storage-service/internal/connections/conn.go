package connections

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitPool(ctx context.Context) *pgxpool.Pool {
	config, err := pgxpool.ParseConfig(getConnectionString())
	if err != nil {
		log.Fatal(err)
	}
	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 6; i++ {
		err = pool.Ping(ctx)
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second)
	}
	if err != nil {
		log.Fatal(err)
	}
	return pool
}

func getConnectionString() string {
	connectionString := os.Getenv("CONNECTION_STRING")
	if connectionString == "" {
		log.Fatal("No connection string found")
	}
	return connectionString
}
