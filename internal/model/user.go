package model

import (
	"golang.org/x/crypto/bcrypt"
)

// User mirrors the Flask User model.
// password_hash is excluded from JSON serialization via the "-" tag.
type User struct {
	ID           int     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username     string  `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	PasswordHash string  `gorm:"type:varchar(256);not null" json:"-"`
	Stores       []Store `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"stores,omitempty"`
}

// UserResponse is the DTO returned by API endpoints, matching
// the Marshmallow UserSchema which excludes password_hash.
type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

// SetPassword hashes the given password using bcrypt and stores it.
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword verifies the given password against the stored hash.
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// ToResponse converts a User to a UserResponse DTO.
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:       u.ID,
		Username: u.Username,
	}
}
