package expect

import (
	"fmt"
	"testing"
)

type TestStruct struct {
	Name string
	Age  int
}

type ExpectStruct struct {
	Name string
	Age  int
}

func (t ExpectStruct) ExpectDisplay() string {
	return fmt.Sprintf("TestStruct{Name: %s, Age: %d}", t.Name, t.Age)
}

func TestExpect(t *testing.T) {
	Test(t, "hey", "hey")
	Test(t, 1, 1)
	Test(t, 100_000, 100000)
	Test(t, TestStruct{}, `
 {
  "Name": "",
  "Age": 0
 }`)

	Test(t, map[string]int{"a": 1, "b": 2}, `
 {
  "a": 1,
  "b": 2
 }`)

	Test(t, ExpectStruct{}, "TestStruct{Name: , Age: 0}")
}
