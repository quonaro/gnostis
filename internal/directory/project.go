package directory

// Project represents an indexed directory root.
type Project struct {
	ID   string
	Name string
	Path string
}

// New creates a project from a directory path and name.
func NewProject(name, path string) Project {
	return Project{
		ID:   name,
		Name: name,
		Path: path,
	}
}
