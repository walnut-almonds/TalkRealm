package model

import "gorm.io/gorm"

// AfterFind 在從 DB 查詢 Message 後，初始化 Attachments 為空切片（確保 JSON 序列化為 [] 而非 null）
func (m *Message) AfterFind(_ *gorm.DB) error {
	if m.Attachments == nil {
		m.Attachments = []string{}
	}

	return nil
}
