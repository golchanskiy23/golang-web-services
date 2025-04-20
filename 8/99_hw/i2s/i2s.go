package main

import (
	"errors"
	"fmt"
	"reflect"
)

func i2s(data interface{}, out interface{}) error {
	outValue := reflect.ValueOf(out)
	if outValue.Kind() != reflect.Ptr {
		return errors.New("data is not a pointer")
	}
	outValue = outValue.Elem()

	switch outValue.Kind() {
	case reflect.Int:
		d, ok := data.(float64)
		if !ok {
			return errors.New("data is not a float64")
		}
		outValue.SetInt(int64(d))
	case reflect.String:
		d, ok := data.(string)
		if !ok {
			return errors.New("data is not a string")
		}
		outValue.SetString(d)
		fmt.Print(outValue)
	case reflect.Bool:
		d, ok := data.(bool)
		if !ok {
			return errors.New("data is not a bool")
		}
		outValue.SetBool(d)
	case reflect.Slice:
		d, ok := data.([]interface{})
		if !ok {
			return errors.New("data is not a []interface{}")
		}
		slice := reflect.MakeSlice(outValue.Type(), len(d), len(d))
		for i := 0; i < len(d); i++ {
			o := reflect.New(outValue.Type().Elem())
			if err := i2s(d[i], o.Interface()); err != nil {
				return fmt.Errorf("Error converting data to slice, %v", err)
			}
			slice.Index(i).Set(o.Elem())
		}
		outValue.Set(slice)
	case reflect.Struct:
		d, ok := data.(map[string]interface{})
		if !ok {
			return errors.New("data is not a map[string]interface{}")
		}
		for i := 0; i < outValue.NumField(); i++ {
			f := outValue.Type().Field(i)
			v, ok := d[f.Name]
			if !ok {
				return errors.New(fmt.Sprintf("Field %v not found in data", f.Name))
			}
			if err := i2s(v, outValue.Field(i).Addr().Interface()); err != nil {
				return errors.New(fmt.Sprintf("Field %v not found in data", f.Name))
			}
		}
	}
	return nil
}
