package builtons

type ObjectRegister interface {
	Name() string
	Object() map[string]interface{}
}
