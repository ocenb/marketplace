package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserPublic struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

type Listing struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ImageURL    string    `json:"image_url"`
	Price       int64     `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
	AuthorLogin string    `json:"author_login"`
	IsOwner     bool      `json:"is_owner"`
}

type ListingsFeed struct {
	Listings []Listing `json:"listings"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	Limit    int       `json:"limit"`
}
