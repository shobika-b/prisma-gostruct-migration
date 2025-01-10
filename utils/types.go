package utils

type Field struct {
	Name          string
	Type          string
	IsDefaultType bool
	annotation    string
}

type Model struct {
	Name   string
	Fields []Field
}

type Enum struct {
	Name   string
	Values []string
}
