package dto

type TurnCredentialsResponse struct {
	URLs     []string `json:"urls"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	TTL      int      `json:"ttl"`
}
