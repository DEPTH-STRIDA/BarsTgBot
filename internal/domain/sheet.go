package domain

import "tg_seller/internal/model"

type SheetService interface {
	InsertClient(row int, client model.Client) error
	FindFirstFreeRow() (int, error)
}
