package entities

import (
	"regexp"
	"strings"
)

type Region struct {
	ID             string
	Name           string
	NormalizedName string
}

func NewRegion(id, name string) Region {
	return Region{
		ID:             id,
		Name:           name,
		NormalizedName: NormalizeRegionName(name),
	}
}

func NormalizeRegionName(name string) string {
	str := strings.ToLower(name)
	str = strings.ReplaceAll(str, "ё", "е")
	str = strings.ReplaceAll(str, "й", "и")

	re := regexp.MustCompile(`[^\wа-яА-Я]+`) // удаление всех символов, кроме букв и цифр
	str = re.ReplaceAllString(str, "")
	return str
}
