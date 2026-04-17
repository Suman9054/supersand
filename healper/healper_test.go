package healper

import (
	"fmt"
	"testing"
)

func TestRandomuuid(t *testing.T){
	id1:=GenrateRandomUUid()
	fmt.Println("id1:",id1)
	id2:=GenrateRandomUUid()
	fmt.Println("id2:",id2)

	if id1==id2{
		t.Fatal("expected different uuids but got same")
	}
}