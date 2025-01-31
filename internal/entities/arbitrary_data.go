package entities

type ArbitraryData struct {
	ID    string `gorm:"primaryKey"`
	Value []byte
}
