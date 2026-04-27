package service

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/che1nov/gopulse/examples/demo-service/internal/repository"
)

func BenchmarkCreateUser(b *testing.B) {
	ctx := context.Background()
	svc := NewUserService(repository.NewMemoryUserRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.CreateUser(ctx, CreateUserInput{
			Name:  "Ada Lovelace",
			Email: "ada@example.com",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateUserParallel(b *testing.B) {
	ctx := context.Background()
	svc := NewUserService(repository.NewMemoryUserRepository(), slog.New(slog.NewTextHandler(io.Discard, nil)))

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := svc.CreateUser(ctx, CreateUserInput{
				Name:  "Grace Hopper",
				Email: "grace@example.com",
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
