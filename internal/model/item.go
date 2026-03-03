package model

// Item mirrors the Flask Item model.
// ItemSchema includes foreign keys (store_id) in serialization.
type Item struct {
	ID      int     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name    string  `gorm:"type:varchar(64);not null" json:"name"`
	Price   float64 `gorm:"not null" json:"price"`
	StoreID int     `gorm:"not null;index" json:"store_id"`
}
