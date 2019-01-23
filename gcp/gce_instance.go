package gcp

import (
	"errors"
	"strings"
	"time"
)

// Given a url for a zone from the api, extract the name of the zone
func zoneFromUrl(url string) (string, error) {
	split := strings.Split(url, "/")
	if len(split) == 0 {
		return "", errors.New("got invalid zone url: " + url)
	}
	return split[len(split)-1], nil
}

// For a given zone, get the region it is located in
func regionFromZone(ctx *GcpContext, zone string) (string, error) {
	for _, region := range ctx.Regions {
		for _, regionZoneUrl := range region.Zones {
			regionZone, err := zoneFromUrl(regionZoneUrl)
			if err != nil {
				return "", err
			}
			if zone == regionZone {
				return region.Name, nil
			}
		}
	}

	return "", errors.New("could not get region for zone: " + zone)
}

// Get the compute instances for the project of the context as a list of
// GcpResources for nuking
func GetAllGceInstances(ctx *GcpContext, excludedRegions []string, excludeAfter time.Time) ([]GcpResource, error) {
	instances := []GcpResource{}

	apiInstances, err := ctx.Service.Instances.AggregatedList(ctx.Project).Do()
	if err != nil {
		return nil, err
	}

	for _, item := range apiInstances.Items {
		for _, apiInstance := range item.Instances {
			// skip if deletion protection is turned on
			if apiInstance.DeletionProtection {
				continue
			}

			zone, err := zoneFromUrl(apiInstance.Zone)
			if err != nil {
				return nil, err
			}

			region, err := regionFromZone(ctx, zone)
			if err != nil {
				return nil, err
			}

			// skip if the region is excluded
			for _, excludedRegion := range excludedRegions {
				if region == excludedRegion {
					continue
				}
			}

			// skip if created after the given time
			creationTime, err := time.Parse(time.RFC3339, apiInstance.CreationTimestamp)
			if err != nil {
				return nil, err
			}
			if creationTime.After(excludeAfter) {
				continue
			}

			instance := GceInstanceResource{
				kind:   apiInstance.Kind,
				name:   apiInstance.Name,
				zone:   zone,
				region: region,
			}

			instances = append(instances, instance)
		}
	}

	return instances, nil
}
