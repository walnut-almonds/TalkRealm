package model

import "gorm.io/gorm"

// AfterFind 在從 DB 查詢 Message 後，確保 Attachments 不為 nil（JSON 序列化為 [] 而非 null）
func (m *Message) AfterFind(_ *gorm.DB) error {
	if m.Attachments == nil {
		m.Attachments = []MessageAttachment{}
	}

	return nil
}
