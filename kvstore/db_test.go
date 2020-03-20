package kvstore

import "testing"

type testObj struct {
	X map[string]string
	Y string
	Z int
}

func TestReadWrite(t *testing.T) {
	db, err := NewDB("keyvalue.db")
	if err != nil {
		t.Error(err)
	}

	testVal := testObj{
		X: map[string]string{"one": "1", "two": "2"},
		Y: "Hello!",
		Z: 42,
	}
	err = db.Table("test").SetObject("12345", testVal)
	if err != nil {
		t.Error(err)
	}

	out := testObj{}
	err = db.Table("test").GetObject("12345", &out)
	if err != nil {
		t.Error(err)
	}
	if out.Y != testVal.Y || out.Z != testVal.Z {
		t.Error("Objects did not match")
	}

	db.Table("test").DeleteObject("12345")
}
