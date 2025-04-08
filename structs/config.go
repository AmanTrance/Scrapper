package structs

type Config struct {
	RabbitURL   string `json:"rabbit_url"`
	PostgresURL string `json:"postgres_url"`
	ServerPort  int    `json:"server_port"`
	Exchange    string `json:"exchange"`
}
