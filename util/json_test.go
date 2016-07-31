package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveFromMap(t *testing.T) {
	m := make(map[string]interface{})
	m["kuku"] = "a"
	m1 := make(map[string]interface{})
	m1["kaka"] = "b"
	m1["kushkush"] = "c"
	m["kiki"] = m1
	m = removeFromMap(m, "kuku")
	if _, has := m["kuku"]; has {
		t.Error("Did not remove first level")
	}
	m = removeFromMap(m, "kiki.kaka")
	m1 = m["kiki"].(map[string]interface{})
	if _, has := m1["kaka"]; has {
		t.Error("Did not remove second level")
	}
	if _, has := m1["kushkush"]; !has {
		t.Error("Deleted too many things")
	}
}

type testT struct {
	Name string    `json:"name"`
	Subs []testSub `json:"subs"`
}

type testSub struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Password string `json:"password"`
}

func TestMarshalWithFilter(t *testing.T) {
	test := testT{Name: "inv"}
	test.Subs = []testSub{{Name: "kuku", IP: "1.2.3.4", Password: "abcd"}, {Name: "kiki", IP: "1.2.3.5", Password: "efgh"}}
	filters := []string{"subs.ip", "subs.password"}
	b, err := MarshalWithFilter(test, filters...)
	if err != nil {
		t.Error(err)
	}
	s := string(b)
	if strings.Contains(s, "abcd") || strings.Contains(s, "efgh") || strings.Contains(s, "1.2.3") {
		t.Error("Did not filter the fields")
	}
	if !strings.Contains(s, "kuku") {
		t.Error("Deleted too many things")
	}
	b, err = MarshalWithFilter([]testT{test, test}, filters...)
	if err != nil {
		t.Error(err)
	}
	s = string(b)
	if strings.Contains(s, "abcd") || strings.Contains(s, "efgh") || strings.Contains(s, "1.2.3") {
		t.Error("Did not filter the fields")
	}
	if !strings.Contains(s, "kuku") {
		t.Error("Deleted too many things")
	}
}

func TestAddPropToJSONString(t *testing.T) {
	s, err := AddPropToJSONString("", "Security.SessionKey", "kuku")
	assert.NoError(t, err)
	assert.Equal(t, `{"Security":{"SessionKey":"kuku"}}`, s)
	s, err = AddPropToJSONString(s, "Security.Some.Other.Key", "kuku")
	assert.NoError(t, err)
	assert.Equal(t, `{"Security":{"SessionKey":"kuku","Some":{"Other":{"Key":"kuku"}}}}`, s)
	s, err = AddPropToJSONString(s, "Top", "key")
	assert.NoError(t, err)
	assert.Equal(t, `{"Security":{"SessionKey":"kuku","Some":{"Other":{"Key":"kuku"}}},"Top":"key"}`, s)
}
