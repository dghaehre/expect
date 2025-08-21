package expect

import (
	"testing"
)

type TestStruct struct {
	Name string
	Age  int
}

func TestExpect(t *testing.T) {
	hey := "hey"
	Equal(t, "hey", "hey")
	Equal(t, hey, "hey")
	heyhey := `
	hey

	hey`
	Equal(t, heyhey, `
	hey

	hey`)
	Equal(t, 1, 1)
	Equal(t, 100_000, 100000)
	Equal(t, 3.14, 3.14)

	JsonEqual(t, TestStruct{}, `
{
  "Name": "",
  "Age": 0
 }`)

	JsonEqual(t, TestStruct{}, `
{
  "Name": "",
  "Age": 0
 }`)

	JsonEqual(t, map[int]string{1: "one", 2: "two"}, `
{
  "1": "one",
  "2": "two"
 }`)

	myMap := map[string]int{"one": 1, "two": 2}
	Equal(t, myMap["one"], 1)
	Equal(t, myMap["two"], 2)

	Equal(t, TestStruct{}.Name, "")
	Equal(t, TestStruct{}.Name, "")
	Equal(t, TestStruct{}.Age, 0)
	Equal(t, nil, nil)

	Equal(t, TestStruct{}.Name, "")
	Equal(t, TestStruct{}.Age, 0)
}

func TestOne(t *testing.T) {
	Equal(t, "one", "one")
}

func TestTwo(t *testing.T) {
	Equal(t, "two", "two")
}
