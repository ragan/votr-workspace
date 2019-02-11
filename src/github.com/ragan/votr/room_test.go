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

func BenchmarkRandString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		randString(10)
	}
}

