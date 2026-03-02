package model

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User mirrors the Flask User model.
// JSON tags match the Marshmallow UserSchema output (excludes password_hash).
type User struct {
	ID           int     `gorm:"primaryKey" json:"id"`
	Username     string  `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string  `gorm:"size:256;not null" json:"-"`
	Stores       []Store `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"stores,omitempty"`
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

// Store mirrors the Flask Store model.
// JSON tags match the Marshmallow StoreSchema output (include_fk = True).
type Store struct {
	ID     int    `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"size:64;not null" json:"name"`
	UserID int    `gorm:"not null;index" json:"user_id"`
	Tags   []Tag  `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"tags,omitempty"`
	Items  []Item `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

// Tag mirrors the Flask Tag model.
// JSON tags match the Marshmallow TagSchema output (include_fk = True).
type Tag struct {
	ID      int    `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"size:64;not null" json:"name"`
	StoreID int    `gorm:"not null;index" json:"store_id"`
}

// Item mirrors the Flask Item model.
// JSON tags match the Marshmallow ItemSchema output (include_fk = True).
type Item struct {
	ID      int     `gorm:"primaryKey" json:"id"`
	Name    string  `gorm:"size:64;not null" json:"name"`
	Price   float64 `gorm:"not null" json:"price"`
	StoreID int     `gorm:"not null;index" json:"store_id"`
}

// AutoMigrateAll runs GORM auto-migration for all models.
// This replaces Flask-SQLAlchemy's db.create_all() / create_db().
func AutoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Store{}, &Tag{}, &Item{})
}
