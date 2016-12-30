package registry

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var errInvalidValue = errors.New("Invalid value")

var (
	tags            = []string{"default", "env", "flag", "registry"}
	typeStringSlice = reflect.TypeOf([]string{})
)

type config struct {
	mutex sync.Locker
	ident string
	items []item
}

func (r *registry) Bind(i sync.Locker) error {
	items, err := bind(i)
	if err != nil {
		return err
	}
	r.configs = append(r.configs, config{
		mutex: i,
		ident: fmt.Sprintf("%s.%d", reflect.TypeOf(i).Elem().Name(), len(r.configs)+1),
		items: items,
	})
	r.bindChan <- struct{}{}
	return nil
}

type item struct {
	key       string
	reference reflect.Value
}

func (i *item) set(rawValue string) error {
	if len(rawValue) == 0 {
		return nil
	}

	var (
		err          error
		defaultValue interface{}
	)

	switch i.reference.Type().Kind() {
	case reflect.String:
		defaultValue = rawValue
	case reflect.Int:
		if defaultValue, err = strconv.ParseInt(rawValue, 10, 0); err == nil {
			defaultValue = int(defaultValue.(int64))
		}
	case reflect.Int8:
		if defaultValue, err = strconv.ParseInt(rawValue, 10, 8); err == nil {
			defaultValue = int8(defaultValue.(int64))
		}
	case reflect.Int16:
		if defaultValue, err = strconv.ParseInt(rawValue, 10, 16); err == nil {
			defaultValue = int16(defaultValue.(int64))
		}
	case reflect.Int32:
		if defaultValue, err = strconv.ParseInt(rawValue, 10, 32); err == nil {
			defaultValue = int32(defaultValue.(int64))
		}
	case reflect.Int64:
		defaultValue, err = strconv.ParseInt(rawValue, 10, 64)
	case reflect.Uint:
		if defaultValue, err = strconv.ParseUint(rawValue, 10, 0); err == nil {
			defaultValue = uint(defaultValue.(uint64))
		}
	case reflect.Uint8:
		if defaultValue, err = strconv.ParseUint(rawValue, 10, 8); err == nil {
			defaultValue = uint8(defaultValue.(uint64))
		}
	case reflect.Uint16:
		if defaultValue, err = strconv.ParseUint(rawValue, 10, 16); err == nil {
			defaultValue = uint16(defaultValue.(uint64))
		}
	case reflect.Uint32:
		if defaultValue, err = strconv.ParseUint(rawValue, 10, 32); err == nil {
			defaultValue = uint32(defaultValue.(uint64))
		}
	case reflect.Uint64:
		defaultValue, err = strconv.ParseUint(rawValue, 10, 64)
	case reflect.Float32:
		if defaultValue, err = strconv.ParseFloat(rawValue, 10); err == nil {
			defaultValue = float32(defaultValue.(float64))
		}
	case reflect.Float64:
		defaultValue, err = strconv.ParseFloat(rawValue, 10)
	case reflect.Bool:
		defaultValue, err = strconv.ParseBool(rawValue)
	case reflect.Slice:
		switch i.reference.Type() {
		case typeStringSlice:
			defaultValue = strings.Split(rawValue, ",")
		}
	}

	if err != nil {
		return err
	}

	if i.reference.CanSet() && defaultValue != nil {
		i.reference.Set(reflect.ValueOf(defaultValue))
	}
	return nil
}

func bind(i interface{}) ([]item, error) {
	var (
		rt = reflect.TypeOf(i)
		rv = reflect.ValueOf(i)
	)

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	var items []item

	for i := 0; i < rt.NumField(); i++ {
		var (
			field = rt.Field(i)
			value = rv.FieldByName(field.Name)
		)
		if len(field.PkgPath) != 0 { // enexported
			continue
		}
		switch field.Type.Kind() {
		case reflect.Struct:
			i, err := bind(value.Addr().Interface())
			if err != nil {
				return nil, fmt.Errorf("'%s': %v", field.Name, err)
			}
			items = append(items, i...)
		default:
			item, err := makeItem(field, value)
			if err != nil {
				return nil, fmt.Errorf("'%s': %v", field.Name, err)
			}
			if len(item.key) != 0 {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

func makeItem(field reflect.StructField, value reflect.Value) (item, error) {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if !value.IsValid() {
		return item{}, errInvalidValue
	}

	var (
		tagValue    string
		rawValue    string
		registryKey string
	)

	for _, key := range tags {
		tagValue = field.Tag.Get(key)
		switch key {
		case "registry":
			registryKey = tagValue
		case "default":
			rawValue = tagValue
		case "env":
			if value := os.Getenv(tagValue); len(value) != 0 {
				rawValue = value
			}
		case "flag":
			if len(tagValue) != 0 {
				if value := flag(tagValue); len(value) != 0 {
					rawValue = value
				}
			}
		}
	}

	new := item{key: registryKey, reference: value}

	if err := new.set(rawValue); err != nil {
		return item{}, err
	}

	return new, nil
}

func flag(name string) string {
	prefix := "--"
	if len(name) == 1 {
		prefix = "-"
	}
	name = prefix + name
	for i, arg := range args {
		if strings.Contains(arg, "=") && strings.HasPrefix(arg, name) {
			if parts := strings.Split(arg, "="); len(parts) == 2 {
				return parts[1]
			}
		}
		if arg == name {
			if len(args) >= i {
				return args[i+1]
			}
		}
	}
	return ""
}
