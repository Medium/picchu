package sentry

import (
	"fmt"
	"time"
)

type Release struct {
	Projects     []Project  `json:"projects,omitempty"`
	DateCreated  *time.Time `json:"dateCreated,omitempty"`
	DateReleased *time.Time `json:"dateReleased,omitempty"`
	Ref          *string    `json:"ref,omitempty"`
	Version      *string    `json:"version"`
}

type Project struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Ref struct {
	Repository     string `json:"repository,omitempty"`
	Commit         string `json:"commit,omitempty"`
	PreviousCommit string `json:"previousCommit,omitempty"`
}

type NewRelease struct {
	Projects []string `json:"projects"`
	Version  string   `json:"version"`
	Ref      string   `json:"ref"`
	Refs     []Ref    `json:"refs,omitempty"`
}

func (c *Client) GetProject(oslug string, pslug string) (Project, error) {
	var proj Project
	err := c.do("GET", fmt.Sprintf("projects/%s/%s", oslug, pslug), &proj, nil)
	return proj, err
}

func (c *Client) CreateProject(oslug string, pslug string) (Project, error) {
	var proj Project
	projreq := &Project{
		Name: pslug,
		Slug: pslug,
	}
	err := c.do("POST", fmt.Sprintf("teams/%s/%s/projects", oslug, "medium"), &proj, projreq)
	return proj, err
}

func (c *Client) CreateRelease(oslug string, r NewRelease) (Release, error) {
	var rel Release
	err := c.do("POST", fmt.Sprintf("organizations/%s/releases", oslug), &rel, &r)
	return rel, err
}
