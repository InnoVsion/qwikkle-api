package auth

import (
	"context"
	"os"

	"golang.org/x/crypto/bcrypt"

	"qwikkle-api/internal/types"
)

func BootstrapAdmin(ctx context.Context, repo Repository) error {
	qkID := os.Getenv("BOOTSTRAP_ADMIN_QKID")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if qkID == "" || password == "" {
		return nil
	}

	normalizedQKID, err := types.NormalizeQKID(qkID)
	if err != nil {
		return err
	}

	_, err = repo.GetUserByQKID(ctx, normalizedQKID)
	if err == nil {
		return nil
	}
	if err != ErrUserNotFound {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = repo.CreateUser(ctx, normalizedQKID, nil, string(hash), string(types.UserRoleAdmin))
	return err
}
