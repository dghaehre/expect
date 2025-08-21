package expect

import (
	"testing"

	"github.com/dghaehre/expect/somepackage"
)

type StringEnum string

type IntEnum int

type FloatEnum float64

const (
	OneIntEnum IntEnum = iota
	TwoIntEnum IntEnum = 2

	OneFloatEnum FloatEnum = 1.0
	TwoFloatEnum FloatEnum = 2.0
)

const (
	One StringEnum = "one"
	Two StringEnum = "two"
)

type TestStruct struct {
	Name   string
	Age    int
	Enum   StringEnum
	Number IntEnum
	Float  FloatEnum
	Result somepackage.Result
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
  "Age": 0,
  "Enum": "",
  "Number": 0,
  "Float": 0,
  "Result": ""
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

	ts := TestStruct{
		Enum:   One,
		Number: OneIntEnum,
		Float:  OneFloatEnum,
		Result: somepackage.Success,
	}
	Equal(t, ts.Enum, StringEnum("one"))
	Equal(t, ts.Number, IntEnum(0))
	Equal(t, ts.Float, FloatEnum(1.000000))
	Equal(t, ts.Result, somepackage.Result("success"))
}

func TestOne(t *testing.T) {
	Equal(t, "one", "one")
}

func TestTwo(t *testing.T) {
	Equal(t, "two", "two")
}
