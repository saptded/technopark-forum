package models

//easyjson:json
type Status struct {
	Forum  int `json:"forum"`
	Post   int `json:"post"`
	Thread int `json:"thread"`
	User   int `json:"user"`
}
