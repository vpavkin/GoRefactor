package refactoring

import (
	"utils"
	"errors"
	"program"
)

const DEFAULT_ORDER string = "cvtmf"
const CONST_ORDER_ENTRY int = 'c'
const VAR_ORDER_ENTRY int = 'v'
const TYPE_ORDER_ENTRY int = 't'
const METHOD_ORDER_ENTRY int = 'm'
const FUNCTION_ORDER_ENTRY int = 'f'

//only correct order input
func getFullOrder(order string) string {
	result := ""
	used := make(map[int]bool)
	for _, b := range order {
		result += string(b)
		used[b] = true
	}
	for _, b := range DEFAULT_ORDER {
		if _, ok := used[b]; !ok {
			result += string(b)
		}
	}
	return result
}

func isOrderEntry(b int) bool {
	switch b {
	case CONST_ORDER_ENTRY:
		fallthrough
	case VAR_ORDER_ENTRY:
		fallthrough
	case TYPE_ORDER_ENTRY:
		fallthrough
	case METHOD_ORDER_ENTRY:
		fallthrough
	case FUNCTION_ORDER_ENTRY:
		return true
	}
	return false
}

func isCorrectOrder(order string) bool {
	used := make(map[int]bool)
	for _, b := range order {
		if !isOrderEntry(b) {
			return false
		}
		if _, ok := used[b]; ok {
			return false
		}
		used[b] = true
	}
	return true
}
func CheckSortParameters(filename string, order string) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case !isCorrectOrder(order):
		return false, errors.ArgumentError("order", "invalid order string, type \"goref sort -h\" to see usage")
	}
	return true, nil
}
func Sort(programTree *program.Program, filename string, groupMethodsByType bool, groupMethodsByVisibility bool, order string) (bool, *errors.GoRefactorError) {

	if ok, err := CheckSortParameters(filename, order); !ok {
		return false, err
	}
	ord := getFullOrder(order)
	println(ord)
	return true, nil
}
