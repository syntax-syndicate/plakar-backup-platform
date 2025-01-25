package fs

import "testing"

func TestPathTmp(t *testing.T) {
	r := &Repository{root: "test"}
	expected := "test/tmp"
	actual := r.PathTmp()
	if actual != expected {
		t.Errorf("PathTmp() = %q, want %q", actual, expected)
	}
}

func TestPathStates(t *testing.T) {
	r := &Repository{root: "test"}
	expected := "test/states"
	actual := r.PathStates()
	if actual != expected {
		t.Errorf("PathStates() = %q, want %q", actual, expected)
	}
}

func TestPathPackfiles(t *testing.T) {
	r := &Repository{root: "test"}
	expected := "test/packfiles"
	actual := r.PathPackfiles()
	if actual != expected {
		t.Errorf("PathPackfiles() = %q, want %q", actual, expected)
	}
}

func TestPathStateBucket(t *testing.T) {
	r := &Repository{root: "test"}
	checksum := [32]byte{0x12}
	expected := "test/states/12"
	actual := r.PathStateBucket(checksum)
	if actual != expected {
		t.Errorf("PathStateBucket() = %q, want %q", actual, expected)
	}
}

func TestPathPackfileBucket(t *testing.T) {
	r := &Repository{root: "test"}
	checksum := [32]byte{0x12}
	expected := "test/packfiles/12"
	actual := r.PathPackfileBucket(checksum)
	if actual != expected {
		t.Errorf("PathPackfileBucket() = %q, want %q", actual, expected)
	}
}

func TestPathState(t *testing.T) {
	r := &Repository{root: "test"}
	checksum := [32]byte{0x12, 0x34, 0x56}
	expected := "test/states/12/1234560000000000000000000000000000000000000000000000000000000000"
	actual := r.PathState(checksum)
	if actual != expected {
		t.Errorf("PathState() = %q, want %q", actual, expected)
	}
}

func TestPathPackfile(t *testing.T) {
	r := &Repository{root: "test"}
	checksum := [32]byte{0x12, 0x34, 0x56}
	expected := "test/packfiles/12/1234560000000000000000000000000000000000000000000000000000000000"
	actual := r.PathPackfile(checksum)
	if actual != expected {
		t.Errorf("PathPackfile() = %q, want %q", actual, expected)
	}
}
