package util

import (
	"math/rand"
	"reflect"
	"sort"
	"strings"

	log "github.com/Sirupsen/logrus"
	stackerr "github.com/go-errors/errors"
)

const (
	// RecoverRoutineForever - this const is used to be passed to RunInSelfRecoverableGoRoutine
	// as maxPanics argument. It indicates that the RunInSelfRecoverableGoRoutine
	// will recover forever from panics
	RecoverRoutineForever = -1
)

type sortedString []string

//Sort functions
func (a sortedString) Len() int           { return len(a) }
func (a sortedString) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortedString) Less(i, j int) bool { return strings.ToLower(a[i]) < strings.ToLower(a[j]) }

// SortString - sort an array of strings
func SortString(arr []string) {
	c := sortedString(arr)
	sort.Sort(c)
}

// Max returns the max between 2 ints
func Max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}

// Min returns the min between 2 ints
func Min(a int, b int) int {
	if a > b {
		return b
	}
	return a
}

// ToIntf converts a slice or array of a specific type to array of interface{}
func ToIntf(s interface{}) []interface{} {
	v := reflect.ValueOf(s)
	// There is no need to check, we want to panic if it's not slice or array
	intf := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		intf[i] = v.Index(i).Interface()
	}
	return intf
}

// RandStr returns a random string of size strSize
func RandStr(strSize int) string {
	dictionary := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	bytes := make([]byte, strSize)
	for k := range bytes {
		v := rand.Int()
		bytes[k] = dictionary[v%len(dictionary)]
	}
	return string(bytes)
}

// ToLower conerts a slice of strings to lower case
func ToLower(s []string) []string {
	res := make([]string, len(s))
	for i, v := range s {
		res[i] = strings.ToLower(v)
	}
	return res
}

// In checks if val is in s slice
func In(slice interface{}, val interface{}) bool {
	si := ToIntf(slice)
	for _, v := range si {
		if v == val {
			return true
		}
	}
	return false
}

// IndexOf returns the index of an object in an array based on obj1 == obj2
func IndexOf(slice interface{}, val interface{}) int {
	si := ToIntf(slice)
	for p, v := range si {
		if v == val {
			return p
		}
	}
	return -1
}

// MapStrings with a translate function (like Array.map in JS)
func MapStrings(arr []string, mapFunc func(int, string) string) []string {
	var res []string
	for i, s := range arr {
		res = append(res, mapFunc(i, s))
	}
	return res
}

// SplitOrEmpty returns the value split by "," or empty array if val is empty
func SplitOrEmpty(val string) (split []string) {
	if len(val) > 0 {
		split = strings.Split(val, ",")
	}
	return
}

// SplitAndTrim , split by token "," and trim rach result
func SplitAndTrim(s string) []string {
	if len(s) == 0 {
		return make([]string, 0)
	}
	arr := strings.Split(s, ",")
	resultArr := make([]string, len(arr))
	for i := range arr {
		resultArr[i] = strings.TrimSpace(arr[i])
	}
	return resultArr
}

// GoAndRespawn - the function runs the f function as a go routine.
// If f panics, it will recover from the panic and re-run the function.
// If f finished without panic, then the go routine will finish.
// maxPanics - if f() paniced maxPanics times, then the routine will be stopped.
//		Use RecoverRoutineForever const to recover forever from panics
// onDone - is a callback which executed when or f() finished gracefully or maxPanics occured.
// 		It passes isFailed which indicates if f() finished gracefully or opaniced maxPanics times
func GoAndRespawn(f func(), maxPanics int, onDone func(isFailed bool)) {
	go func() {
		// if the panicsLeft is positive then after panicsLeft panics it will reduce to 0
		// if the panicsLeft is negative then it will never be 0, so the for run forever
		panicsLeft := maxPanics
		for fPaniced := true; fPaniced && panicsLeft != 0; {
			fPaniced = func() (paniced bool) {
				defer func() {
					if err := recover(); err != nil {
						log.Error(err)
						log.Error(stackerr.Wrap(err, 2).ErrorStack())
						panicsLeft--
						paniced = true
					}
				}()
				f()

				// f() function finished without panic, 'paniced' is false
				return
			}()
		}

		if onDone != nil {
			onDone(panicsLeft == 0)
		}
	}()
}
