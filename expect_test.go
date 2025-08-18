package expect

import (
	"fmt"
	"testing"
)

type TestStruct struct {
	Name string
	Age  int
}

func (t TestStruct) String() string {
	return fmt.Sprintf("TestStruct{Name: %s, Age: %d}", t.Name, t.Age)
}

func TestExpect(t *testing.T) {

	Test(t, "hey", "hey")
	Test(t, 1, 1)
	Test(t, 100_000, 100000)
	// Test(t, map[string]int{"a": 1, "b": 2})
	Test(t, TestStruct{}, "TestStruct{Name: , Age: 0}")

}
