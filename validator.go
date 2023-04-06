package validator

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")
var ErrFieldNotValid = errors.New("field not valid")

type ValidationError struct {
	Err error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var sb strings.Builder

	for _, err := range v {
		sb.WriteString(err.Err.Error())
	}
	return sb.String()
}

func Validate(v any) error {
	var allErrors ValidationErrors

	valueV := reflect.ValueOf(v)
	typeV := reflect.TypeOf(v)

	if typeV.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	for i := 0; i < typeV.NumField(); i++ {
		validCond := typeV.Field(i).Tag.Get("validate")
		if len(validCond) == 0 {
			continue
		}

		if !typeV.Field(i).IsExported() {
			allErrors = append(allErrors, ValidationError{ErrValidateForUnexportedFields})
			continue
		}

		validator, errParse := parseValidators(validCond)
		if errParse != nil {
			allErrors = append(allErrors, ValidationError{errParse})
			continue
		}

		kind := typeV.Field(i).Type.Kind()

		var err error

		switch kind {
		case reflect.Slice:
			err = validateSlice(validator, valueV.Field(i))
		default:
			err = validateValue(validator, kind, valueV.Field(i))
		}

		if err != nil {
			allErrors = append(allErrors, ValidationError{errors.New("field: " + typeV.Field(i).Name + " not valid for " + validCond)})
		}
	}
	if len(allErrors) == 0 {
		return nil
	}
	return allErrors
}

func parseValidators(get string) ([]Validator, error) {
	parts := strings.Split(get, "&")
	var allValidators []Validator
	for _, cond := range parts {
		validator, err := parseValidator(cond)
		if err != nil {
			return nil, err
		}
		allValidators = append(allValidators, validator)
	}
	return allValidators, nil
}

func validateSlice(validators []Validator, value reflect.Value) error {
	for _, validator := range validators {
		for i := 0; i < value.Len(); i++ {
			err := validateValue([]Validator{validator}, value.Index(i).Kind(), value.Index(i))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func validateValue(validators []Validator, kind reflect.Kind, field reflect.Value) error {
	var err error
	for _, validator := range validators {
		switch validator.name {
		case "len":
			if kind == reflect.String {
				err = validateLen(field.String(), validator.argsInt[0])
			} else {
				err = ErrFieldNotValid
			}
		case "min":
			switch kind {
			case reflect.String:
				err = validateMin(len(field.String()), validator.argsInt[0])
			case reflect.Int:
				err = validateMin(int(field.Int()), validator.argsInt[0])
			default:
				err = ErrFieldNotValid
			}
		case "max":
			switch kind {
			case reflect.String:
				err = validateMax(len(field.String()), validator.argsInt[0])
			case reflect.Int:
				err = validateMax(int(field.Int()), validator.argsInt[0])
			default:
				err = ErrFieldNotValid
			}
		case "in":
			switch kind {
			case reflect.String:
				err = validateIn(field.String(), validator.argsStr)
			case reflect.Int:
				err = validateIn(int(field.Int()), validator.argsInt)
			default:
				err = ErrFieldNotValid
			}
		default:
			err = ErrInvalidValidatorSyntax
		}

		if err != nil {
			return ErrFieldNotValid
		}
	}
	return nil
}

type Validator struct {
	name    string
	argsStr []string
	argsInt []int
}

func parseValidator(get string) (Validator, error) {
	parts := strings.Split(get, ":")
	name := strings.TrimSpace(parts[0])
	argsStr := strings.Split(parts[1], ",")
	var args []int
	for _, arg := range argsStr {
		num, err := strconv.Atoi(strings.TrimSpace(arg))
		if err != nil && name != "in" {
			return Validator{}, ErrInvalidValidatorSyntax
		}
		args = append(args, num)
	}

	if name == "in" && len(argsStr) == 1 && argsStr[0] == "" {
		argsStr = []string{}
	}

	return Validator{
		name:    name,
		argsStr: argsStr,
		argsInt: args,
	}, nil
}

func validateLen(field string, num int) error {
	if len(field) == num {
		return nil
	}
	return ErrFieldNotValid
}

func validateMin(field int, num int) error {
	if field >= num {
		return nil
	}

	return ErrFieldNotValid
}

func validateMax(field int, num int) error {
	if field <= num {
		return nil
	}

	return ErrFieldNotValid
}

func validateIn[T comparable](field T, args []T) error {
	for _, v := range args {
		if v == field {
			return nil
		}
	}
	return ErrFieldNotValid
}
