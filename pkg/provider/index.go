package provider

type InstancesProvider interface {
	Init(map[string]interface{}) error
	GetAllInstances() []string
	GetActiveInstances() []string
}
