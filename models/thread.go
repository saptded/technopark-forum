package models

import "time"

//easyjson:json
type Thread struct {
	ID      int       `json:"id"`
	Title   string    `json:"title"`
	Author  string    `json:"author"`
	Forum   string    `json:"forum"`
	Message string    `json:"message"`
	Votes   int       `json:"votes"`
	Slug    string    `json:"slug"`
	Created time.Time `json:"created"`
}

//easyjson:json
type Threads []Thread

//easyjson:json
type ThreadUpdate struct {
	Message *string `json:"message"`
	Title   *string `json:"title"`
}
