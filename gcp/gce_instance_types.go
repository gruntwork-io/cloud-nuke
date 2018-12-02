package gcp

type GceInstanceResource struct {
	InstanceName string
	Zone         string
	RegionName   string
}

func (instance GceInstanceResource) ResourceName() string {
	return "GCE Instance"
}

func (instance GceInstanceResource) Name() string {
	return instance.InstanceName
}

func (instance GceInstanceResource) LocationName() string {
	return "Zone"
}

func (instance GceInstanceResource) Location() string {
	return instance.Zone
}

func (instance GceInstanceResource) Region() string {
	return instance.RegionName
}

func (instance GceInstanceResource) Nuke(ctx *GcpContext) error {
	_, err := ctx.Service.Instances.Delete(ctx.Project, instance.Zone, instance.InstanceName).Do()
	return err
}
