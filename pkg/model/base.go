package model

import "time"

// BaseRecordFields tracks who created and last updated a record, including IP addresses.
// Embed this into any domain model that requires audit trail information.
type BaseRecordFields struct {
	CreatedAt        time.Time `json:"created_at,omitempty"  gorm:"autoCreateTime"`
	CreatedBy        string    `json:"created_by,omitempty"  gorm:"type:varchar(36)"`
	CreatedByName    string    `json:"created_by_name,omitempty"`
	CreatedIpAddress string    `json:"created_ip_address,omitempty"`

	UpdatedAt        time.Time `json:"updated_at,omitempty"  gorm:"autoUpdateTime"`
	UpdatedBy        string    `json:"updated_by,omitempty"  gorm:"type:varchar(36)"`
	UpdatedByName    string    `json:"updated_by_name,omitempty"`
	UpdatedIpAddress string    `json:"updated_ip_address,omitempty"`
}

// SetCreatedUser populates creation audit fields. Call this before inserting a new record.
func (b *BaseRecordFields) SetCreatedUser(id, name, ipAddress string) {
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	b.CreatedBy = id
	b.CreatedByName = name
	b.CreatedIpAddress = ipAddress
	b.UpdatedAt = b.CreatedAt
	b.UpdatedBy = ""
	b.UpdatedByName = ""
	b.UpdatedIpAddress = ""
}

// SetUpdatedUser populates update audit fields. Call this before saving changes to a record.
func (b *BaseRecordFields) SetUpdatedUser(id, name, ipAddress string) {
	b.UpdatedAt = time.Now()
	b.UpdatedBy = id
	b.UpdatedByName = name
	b.UpdatedIpAddress = ipAddress
}
