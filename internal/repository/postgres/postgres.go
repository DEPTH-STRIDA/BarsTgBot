package postgres

import (
	"tg_seller/internal/model"

	"gorm.io/gorm"
)

type ClientRepository struct {
	DB *gorm.DB
}

func NewClientRepository(db *gorm.DB) *ClientRepository {
	return &ClientRepository{DB: db}
}

// Вставка клиента
func (r *ClientRepository) InsertClient(client *model.Client) error {
	return r.DB.Create(client).Error
}

// Получение всех клиентов с SheetIsSynced=false
func (r *ClientRepository) GetUnsyncedClients() ([]model.Client, error) {
	var clients []model.Client
	err := r.DB.Where("sheet_is_synced = ?", false).Find(&clients).Error
	return clients, err
}

// Обновление поля SheetIsSynced по id
func (r *ClientRepository) UpdateSheetIsSynced(id uint, synced bool) error {
	return r.DB.Model(&model.Client{}).Where("id = ?", id).Update("sheet_is_synced", synced).Error
}

// Проверка существования клиента по телефону и бару
func (r *ClientRepository) ExistsByPhoneAndBar(phone string, bar string) (bool, error) {
	var count int64
	err := r.DB.Model(&model.Client{}).
		Where("phone = ? AND bar = ?", phone, bar).
		Count(&count).Error
	return count > 0, err
}
