package model

// Tag mirrors the Flask Tag model.
// TagSchema includes foreign keys (store_id) in serialization.
type Tag struct {
	ID      int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name    string `gorm:"type:varchar(64);not null" json:"name"`
	StoreID int    `gorm:"not null;index" json:"store_id"`
}
