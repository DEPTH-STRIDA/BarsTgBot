package domain

import "tg_seller/internal/model"

type UserRepo interface {
	// Вставка клиента
	InsertClient(client *model.Client) error

	// Получение всех клиентов с SheetIsSynced=false
	GetUnsyncedClients() ([]model.Client, error)

	// Обновление поля SheetIsSynced по id
	UpdateSheetIsSynced(id uint, synced bool) error

	// Проверка существования клиента по телефону и бару
	ExistsByPhoneAndBar(phone string, bar string) (bool, error)
}
