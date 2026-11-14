package directory

import "testing"

func TestNew(t *testing.T) {
	p := NewProject("myproj", "/tmp/myproj")
	if p.ID != "myproj" {
		t.Errorf("ID = %q, want %q", p.ID, "myproj")
	}
	if p.Name != "myproj" {
		t.Errorf("Name = %q, want %q", p.Name, "myproj")
	}
	if p.Path != "/tmp/myproj" {
		t.Errorf("Path = %q, want %q", p.Path, "/tmp/myproj")
	}
}
