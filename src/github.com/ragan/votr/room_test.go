package votr

import "testing"

func TestNewRoom(t *testing.T) {
	NewRoom()
	NewRoom()
	expLen := 2
	if len(rooms) != expLen {
		t.Errorf("Rooms count should be %d", expLen)
	}
}

