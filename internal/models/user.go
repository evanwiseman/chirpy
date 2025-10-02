package models

import (
	"encoding/json"
	"time"

	"github.com/evanwiseman/chirpy/internal/database"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func FormatUser(u database.User) (User, error) {
	return User{
		ID:        u.ID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		Email:     u.Email,
	}, nil
}

func FormatUserWithToken(u database.User, token string) (map[string]any, error) {
	// Get the current format of the user
	user, err := FormatUser(u)
	if err != nil {
		return nil, err
	}

	// Convert struct to map
	userMap := make(map[string]any)
	dataBytes, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	// unpack user into a json and append the token
	if err := json.Unmarshal(dataBytes, &userMap); err != nil {
		return nil, err
	}

	userMap["token"] = token
	return userMap, nil
}
