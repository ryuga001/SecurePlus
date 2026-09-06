package db

import "gorm.io/gorm"

func TenantScope(customerID int) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		return tx.Where("customer_id = ?", customerID)
	}
}
