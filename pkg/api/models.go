package api

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

type Profile struct {
	Name        string            `json:"name"`
	Tagline     string            `json:"tagline"`
	Bio         string            `json:"bio"`
	Title       string            `json:"title"`
	Location    string            `json:"location"`
	Status      string            `json:"status"`
	Email       string            `json:"email"`
	Phone       string            `json:"phone"`
	Avatar      string            `json:"avatar"`
	Stats       []StatMetric      `json:"stats"`
	SocialLinks map[string]string `json:"social_links"`
	Highlights  []string          `json:"highlights"`
}

type StatMetric struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Sub   string `json:"sub"`
}

type Project struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Tagline     string   `json:"tagline"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Featured    bool     `json:"featured"`
	GithubURL   string   `json:"github_url"`
	DemoURL     string   `json:"demo_url"`
	Image       string   `json:"image"`
	Metrics     string   `json:"metrics"`
}

type SkillItem struct {
	Name        string `json:"name"`
	Level       int    `json:"level"` // 1-100
	Proficiency string `json:"proficiency"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
}

type SkillCategory struct {
	Category string      `json:"category"`
	Summary  string      `json:"summary"`
	Skills   []SkillItem `json:"skills"`
}

type ContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (c *ContactRequest) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(c.Email) == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(c.Email); err != nil {
		return errors.New("invalid email address format")
	}
	if strings.TrimSpace(c.Message) == "" {
		return errors.New("message is required")
	}
	return nil
}

type ContactSubmission struct {
	ID        string    `json:"id"`
	Contact   ContactRequest `json:"contact"`
	CreatedAt time.Time `json:"created_at"`
}
