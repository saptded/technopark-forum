package models

//easyjson:json
type Forum struct {
	Title   string `json:"title"`
	Author  string `json:"user"`
	Slug    string `json:"slug"`
	Posts   int    `json:"posts"`
	Threads int    `json:"threads"`
}
