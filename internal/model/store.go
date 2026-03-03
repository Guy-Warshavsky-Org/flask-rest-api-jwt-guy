package model

// Store mirrors the Flask Store model.
// StoreSchema includes foreign keys (user_id, store_id) in serialization.
type Store struct {
	ID     int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name   string `gorm:"type:varchar(64);not null" json:"name"`
	UserID int    `gorm:"not null;index" json:"user_id"`
	Tags   []Tag  `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"tags,omitempty"`
	Items  []Item `gorm:"foreignKey:StoreID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}
