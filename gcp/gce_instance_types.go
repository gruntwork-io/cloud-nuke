package gcp

type GceInstanceResource struct {
	kind   string
	name   string
	zone   string
	region string
}

func (instance GceInstanceResource) Kind() string {
	return instance.kind
}

func (instance GceInstanceResource) Name() string {
	return instance.name
}

func (instance GceInstanceResource) Zone() string {
	return instance.zone
}

func (instance GceInstanceResource) Region() string {
	return instance.region
}

func (instance GceInstanceResource) Nuke(ctx *GcpContext) error {
	_, err := ctx.Service.Instances.Delete(ctx.Project, instance.zone, instance.name).Do()
	return err
}
