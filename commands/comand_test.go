package comands

import (
	"testing"

	"github.com/suman9054/supersand/store"
)

func TestCreatuser(t *testing.T) {
	s := store.Newstore()
	c := newcomands(s)
	if err := c.CreatNewuser(user{username: "suman", password: "hazra123"}); err != nil {
		t.Fatalf("creatuser: %v", err)
	}
}
