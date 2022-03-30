package errors

import "fmt"

type NotFoundError struct {
	Type string
	Name string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s %s not found", e.Type, e.Name)
}

func NotFoundErr(t, name string) NotFoundError {
	return NotFoundError{
		Type: t,
		Name: name,
	}
}

type NotUniqueError struct {
	Type string
	Name string
}

func (e NotUniqueError) Error() string {
	return fmt.Sprintf("multiple %s %s exist", e.Type, e.Name)
}

func NotUniqueErr(t, name string) NotUniqueError {
	return NotUniqueError{
		Type: t,
		Name: name,
	}
}
