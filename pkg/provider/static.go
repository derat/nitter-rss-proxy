package provider

type StaticProvider struct {
	instance []string
}

func NewStaticProvider() InstancesProvider {
	return &StaticProvider{}
}

func (p *StaticProvider) Init(cfg map[string]interface{}) error {
	if cfg != nil {
		if instance, ok := cfg["instance"]; ok {
			p.instance = instance.([]string)
		}
	}
	return nil
}

func (p *StaticProvider) GetAllInstances() []string {
	return p.instance
}
func (p *StaticProvider) GetActiveInstances() []string {
	return p.instance
}
