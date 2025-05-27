package config

type Config struct {
	TelegramConfig
	DBConfig
	GoogleSheetConfig
}
type GoogleSheetConfig struct {
	SheetID           string `envconfig:"SHEET_ID" required:"true" masked:"true"`
	ClientListID      string `envconfig:"CLIENT_LIST_ID" required:"true" masked:"true"`
	CredentialsBase64 string `envconfig:"CREDENTIALS_BASE64" required:"true" masked:"true"`
	PauseMs           int    `envconfig:"SHEET_PAUSE_MS" required:"false"`
}

type TelegramConfig struct {
	BotToken string `envconfig:"BOT_TOKEN" required:"true" masked:"true"`
	Admins   string `envconfig:"ADMINS" required:"true" masked:"true"`
}

type DBConfig struct {
	User   string `envconfig:"DBUSER" required:"true" masked:"true"`
	Pass   string `envconfig:"DBPASS" required:"true" masked:"true"`
	Host   string `envconfig:"DBHOST" required:"true" masked:"true"`
	DBName string `envconfig:"DBNAME" required:"true" masked:"true"`

	Port    string `envconfig:"DBPORT" required:"true" masked:"true"`
	SSLMode string `envconfig:"DBSSLMODE" required:"true" masked:"true"`
}
