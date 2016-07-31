package util

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToIntf(t *testing.T) {
	if reflect.TypeOf(ToIntf([]int{1, 2, 3})) != reflect.TypeOf([]interface{}{}) {
		t.Fatal("Did not convert slice to intf slice")
	}
}

func TestIn(t *testing.T) {
	s := []string{"foo", "bar", "kuku", "kiki"}
	for _, v := range s {
		if !In(s, v) {
			t.Error("Should be in")
		}
	}
	if In(s, "foobar") {
		t.Error("Should not be in")
	}
}

func TestInStringSlice(t *testing.T) {
	s := []string{"fOo", "baR", "Kuku", "kIki"}
	for _, v := range s {
		if !In(s, v) {
			t.Error("Should be in")
		}
	}
	if In(s, "foobar") {
		t.Error("Should not be in")
	}
}

func TestToLower(t *testing.T) {
	s := []string{"MyString12Str"}
	res := ToLower(s)
	if res[0] != "mystring12str" {
		t.Error(res)
	}

}

func TestRandStr(t *testing.T) {
	s1 := RandStr(32)
	s2 := RandStr(32)
	if len(s1) != 32 {
		t.Errorf("Rand str len not enforced s1 %d", len(s1))
	}
	if len(s2) != 32 {
		t.Errorf("Rand str len not enforced s2 %d", len(s2))
	}
	if s1 == s2 {
		t.Error("Random string is not random")
	}

}

func TestMax(t *testing.T) {
	max := 80
	if Max(1, max) != max {
		t.Errorf("Max value expected to be %d", max)
	}

	max = -1
	if Max(max, -10) != max {
		t.Errorf("Max value expected to be %d", max)
	}
}

func TestMin(t *testing.T) {
	min := 80
	if Min(100, min) != min {
		t.Errorf("Max value expected to be %d", min)
	}

	min = -1
	if Min(min, 0) != min {
		t.Errorf("Max value expected to be %d", min)
	}
}

func TestMapStrings(t *testing.T) {
	in := []string{"a", "b", "c"}
	out := []string{"a0", "b1", "c2"}
	assert.Equal(t, MapStrings(in, func(i int, s string) string {
		return fmt.Sprintf("%s%d", s, i)
	}), out, "Did not map func correctly")
}

func TestIndexOf(t *testing.T) {
	s := []int{101, 19, 73}
	i := IndexOf(s, 19)
	j := IndexOf(s, 7)
	assert.Equal(t, 1, i)
	assert.Equal(t, -1, j)
}

func TestSplitOrEmpty(t *testing.T) {
	assert.Empty(t, SplitOrEmpty(""), "Should be empty array")
	assert.Len(t, SplitOrEmpty("x,y,z"), 3, "Now we are testing split")
}

func TestSplitAndTrim(t *testing.T) {
	assert.Empty(t, SplitAndTrim(""), "Should be empty array")
	assert.EqualValues(t, SplitAndTrim(" a,b ,c,d d"), []string{"a", "b", "c", "d d"}, "Now we are testing split")
	assert.EqualValues(t, 4, len(SplitAndTrim(" a,b ,c,d d")))
}

func TestSort(t *testing.T) {
	s := []string{"c", "b", "A"}
	SortString(s)
	assert.EqualValues(t, []string{"A", "b", "c"}, s, "array not sroted properly")
}

func TestGoAndRespawnRunning3Times(t *testing.T) {
	assert := assert.New(t)
	panicCount := 3
	onError := make(chan bool)
	GoAndRespawn(func() {
		if panicCount > 0 {
			panicCount--
			panic("justforfun.panicCountleft:" + strconv.Itoa(panicCount))
		}
		assert.Fail("thismethodshouldnotberunning")
	}, panicCount, func(isFailed bool) {
		if isFailed == true {
			close(onError)
		}
	})

	select {
	case <-onError:
	case <-time.After(3 * time.Second):
		t.Fatal("onFinishshouldbecalledwithisFailed=true")
	}
}

func TestGoAndRespawnRunningForever(t *testing.T) {
	panicsLeft := 6
	enough := make(chan bool)
	GoAndRespawn(func() {
		if panicsLeft == 0 {
			close(enough)
		}
		panicsLeft--
		panic("justforfun.panicsleft:" + strconv.Itoa(panicsLeft))
	}, RecoverRoutineForever, func(isFailed bool) {
		t.Fatal("GoAndRespawnshouldrecoverforever")
	})

	select {
	case <-enough:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout3seconds.GoAndRespawnshouldrecoverfrompanic6timestillnow!")
	}
}
