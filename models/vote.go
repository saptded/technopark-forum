package models

//easyjson:json
type Vote struct {
	Nickname string `json:"nickname"`
	Voice    int    `json:"voice"`
}

//easyjson:json
type VoteDB struct {
	ID       int
	Nickname string
	ThreadID int
	Voice    int
}
