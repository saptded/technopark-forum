package models

//easyjson:json
type User struct {
	Email    string `json:"email"`
	Nickname string `json:"nickname,omitempty"`
	Fullname string `json:"fullname"`
	About    string `json:"about,omitempty"`
}

//easyjson:json
type Users []User
