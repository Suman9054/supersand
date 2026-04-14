package store

import (
	"testing"
	"time"
)

func TestCashSetGetUpdateAndDelete(t *testing.T) {
	s := Newstore()

	now := time.Now()
   
	s.Chash.Set("suman", Userdata{
		Id:            "suman",
		Useuniqename:  "suman9054",
		
		Lastacces:     now,
		Processstatus: Active,
	})

	data, ok := s.Chash.Get("suman")
	if !ok {
		t.Fatal("expected to get the value but got nothing")
	}

	if data.Id != "suman" || data.Useuniqename != "suman9054" {
		t.Fatal("wrong data returned")
	}

	expected := now.Add(10 * time.Minute)


	err, ok := s.Chash.Update("suman", func(u Userdata) Userdata {
		u.Lastacces = expected
		return u
	})

	if err != nil || !ok {
		t.Fatal("expected update to succeed")
	}

	data, ok = s.Chash.Get("suman")
	if !ok {
		t.Fatal("expected to get updated value")
	}

	if !data.Lastacces.Equal(expected) {
		t.Fatal("Lastacces was not updated correctly")
	}

	if !s.Chash.Remove("suman") {
		t.Fatal("expected remove to succeed")
	}

	_, ok = s.Chash.Get("suman")
	if ok {
		t.Fatal("expected value to be deleted")
	}
}