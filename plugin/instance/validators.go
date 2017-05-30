package instance

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

var numericZeros = []interface{}{
	int(0),
	int8(0),
	int16(0),
	int32(0),
	int64(0),
	uint(0),
	uint8(0),
	uint16(0),
	uint32(0),
	uint64(0),
	float32(0),
	float64(0),
}

// isEmpty is copied from github.com/stretchr/testify/assert/assetions.go
func isEmpty(object interface{}) bool {

	if object == nil {
		return true
	} else if object == "" {
		return true
	} else if object == false {
		return true
	}

	for _, v := range numericZeros {
		if object == v {
			return true
		}
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	case reflect.Map:
		fallthrough
	case reflect.Slice, reflect.Chan:
		{
			return (objValue.Len() == 0)
		}
	case reflect.Struct:
		switch object.(type) {
		case time.Time:
			return object.(time.Time).IsZero()
		}
	case reflect.Ptr:
		{
			if objValue.IsNil() {
				return true
			}
			switch object.(type) {
			case *time.Time:
				return object.(*time.Time).IsZero()
			default:
				return false
			}
		}
	}
	return false
}

func validateSakuraID(fieldName string, object interface{}) []error {
	res := []error{}
	idLen := 12

	// if target is nil , return OK(Use required attr if necessary)
	if object == nil {
		return res
	}

	if id, ok := object.(int64); ok {
		if id == 0 {
			return res
		}
		s := fmt.Sprintf("%d", id)
		strlen := utf8.RuneCountInString(s)
		if id < 0 || strlen != idLen {
			res = append(res, fmt.Errorf("%q: Resource ID must be a %d digits number", fieldName, idLen))
		}
	}

	return res
}

func validateInStrValues(fieldName string, object interface{}, allows ...string) []error {
	res := []error{}

	// if target is nil , return OK(Use required attr if necessary)
	if object == nil {
		return res
	}

	if v, ok := object.(string); ok {
		if v == "" {
			return res
		}

		exists := false
		for _, allow := range allows {
			if v == allow {
				exists = true
				break
			}
		}
		if !exists {
			err := fmt.Errorf("%q: must be in [%s]", fieldName, strings.Join(allows, ","))
			res = append(res, err)
		}
	}
	return res
}

func validateRequired(fieldName string, object interface{}) []error {
	if isEmpty(object) {
		return []error{fmt.Errorf("%q: is required", fieldName)}
	}
	return []error{}
}

func validateSetProhibited(fieldName string, object interface{}) []error {
	if !isEmpty(object) {
		return []error{fmt.Errorf("%q: can't set on current context", fieldName)}
	}
	return []error{}
}

func validateConflicts(fieldName string, object interface{}, values map[string]interface{}) []error {

	if !isEmpty(object) {
		for _, v := range values {
			if !isEmpty(v) {
				keys := []string{}
				for k := range values {
					keys = append(keys, fmt.Sprintf("%q", k))
				}
				return []error{fmt.Errorf("%q: is conflict with %s", fieldName, strings.Join(keys, " or "))}
			}
		}
	}
	return []error{}

}

func validateConflictValues(fieldName string, object interface{}, values map[string]interface{}) []error {

	if !isEmpty(object) {
		for _, v := range values {
			if !isEmpty(v) {
				keys := []string{}
				for k := range values {
					keys = append(keys, fmt.Sprintf("%q", k))
				}
				return []error{fmt.Errorf("%q(%#v): is conflict with %s", fieldName, object, strings.Join(keys, " or "))}
			}
		}
	}
	return []error{}

}

func validateBetween(fieldName string, object interface{}, min int, max int) []error {

	if object == nil {
		object = []int64{}
	}

	isSlice := func(object interface{}) bool {
		_, ok1 := object.([]int64)
		_, ok2 := object.([]string)

		return ok1 || ok2
	}

	if isSlice(object) {
		sliceLen := 0
		if s, ok := object.([]int64); ok {
			sliceLen = len(s)
		} else {
			s := object.([]string)
			sliceLen = len(s)
		}

		if max <= 0 {
			if sliceLen < min {
				return []error{fmt.Errorf("%q: slice length must be %d or more", fieldName, min)}
			}
		} else {
			if !(min <= sliceLen && sliceLen <= max) {
				return []error{fmt.Errorf("%q: slice length must be beetween %d and %d", fieldName, min, max)}
			}

		}
	}

	return []error{}
}
