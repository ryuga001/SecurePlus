package db

import (
	"errors"

	"gorm.io/gorm"
)

func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func IsDuplicate(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

func IsMissingReference(err error) bool {
	return errors.Is(err, gorm.ErrForeignKeyViolated)
}
