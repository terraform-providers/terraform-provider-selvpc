package selectel

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/helper/hashcode"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	v1 "github.com/selectel/mks-go/pkg/v1"
	"github.com/selectel/mks-go/pkg/v1/cluster"
	"github.com/selectel/mks-go/pkg/v1/kubeversion"
	"github.com/selectel/mks-go/pkg/v1/node"
	"github.com/selectel/mks-go/pkg/v1/nodegroup"
)

const (
	ru1Region = "ru-1"
	ru2Region = "ru-2"
	ru3Region = "ru-3"
	ru7Region = "ru-7"
	ru8Region = "ru-8"

	ru1MKSClusterV1Endpoint = "https://ru-1.mks.selcloud.ru/v1"
	ru2MKSClusterV1Endpoint = "https://ru-2.mks.selcloud.ru/v1"
	ru3MKSClusterV1Endpoint = "https://ru-3.mks.selcloud.ru/v1"
	ru7MKSClusterV1Endpoint = "https://ru-7.mks.selcloud.ru/v1"
	ru8MKSClusterV1Endpoint = "https://ru-8.mks.selcloud.ru/v1"
)

func getMKSClusterV1Endpoint(region string) (endpoint string) {
	switch region {
	case ru1Region:
		endpoint = ru1MKSClusterV1Endpoint
	case ru2Region:
		endpoint = ru2MKSClusterV1Endpoint
	case ru3Region:
		endpoint = ru3MKSClusterV1Endpoint
	case ru7Region:
		endpoint = ru7MKSClusterV1Endpoint
	case ru8Region:
		endpoint = ru8MKSClusterV1Endpoint
	}

	return
}

func hashMKSClusterNodegroupsV1(v interface{}) int {
	m := v.(map[string]interface{})
	return hashcode.String(fmt.Sprintf("%s-", m["name"].(string)))
}

func expandMKSClusterNodegroupsV1CreateOpts(v interface{}) *nodegroup.CreateOpts {
	m := v.(map[string]interface{})
	opts := nodegroup.CreateOpts{}
	if _, ok := m["count"]; ok {
		opts.Count = m["count"].(int)
	}
	if _, ok := m["flavor_id"]; ok {
		opts.FlavorID = m["flavor_id"].(string)
	}
	if _, ok := m["cpus"]; ok {
		opts.CPUs = m["cpus"].(int)
	}
	if _, ok := m["ram_mb"]; ok {
		opts.RAMMB = m["ram_mb"].(int)
	}
	if _, ok := m["volume_gb"]; ok {
		opts.VolumeGB = m["volume_gb"].(int)
	}
	if _, ok := m["volume_type"]; ok {
		opts.VolumeType = m["volume_type"].(string)
	}
	if _, ok := m["keypair_name"]; ok {
		opts.KeypairName = m["keypair_name"].(string)
	}
	if _, ok := m["affinity_policy"]; ok {
		opts.AffinityPolicy = m["affinity_policy"].(string)
	}
	if _, ok := m["availability_zone"]; ok {
		opts.AvailabilityZone = m["availability_zone"].(string)
	}

	return &opts
}

func waitForMKSClusterV1ActiveState(
	ctx context.Context, client *v1.ServiceClient, clusterID string, timeout time.Duration) error {
	pending := []string{
		string(cluster.StatusPendingCreate),
		string(cluster.StatusPendingUpdate),
		string(cluster.StatusPendingUpgradePatchVersion),
		string(cluster.StatusPendingResize),
	}
	target := []string{
		string(cluster.StatusActive),
	}

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    mksClusterV1StateRefreshFunc(ctx, client, clusterID),
		Timeout:    timeout,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"error waiting for the cluster %s to become 'ACTIVE': %s",
			clusterID, err)
	}

	return nil
}

func mksClusterV1StateRefreshFunc(
	ctx context.Context, client *v1.ServiceClient, clusterID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		c, _, err := cluster.Get(ctx, client, clusterID)
		if err != nil {
			return nil, "", err
		}

		return c, string(c.Status), nil
	}
}

func flattenMKSClusterNodegroupsV1(d *schema.ResourceData, views []*nodegroup.View) []interface{} {
	nodegroups := d.Get("nodegroups").(*schema.Set).List()
	for i, v := range nodegroups {
		m := v.(map[string]interface{})
		if _, ok := m["id"]; ok {
			for _, view := range views {
				if m["id"] == view.ID {
					m["flavor_id"] = view.FlavorID
					m["volume_gb"] = view.VolumeGB
					m["volume_type"] = view.VolumeType
					m["local_volume"] = view.LocalVolume
					m["availability_zone"] = view.AvailabilityZone
					m["nodes"] = flattenMKSClusterNodesV1(view.Nodes)
				}
				nodegroups[i] = m
			}
		}
	}

	return nodegroups
}

func flattenMKSClusterNodesV1(views []*node.View) []map[string]interface{} {
	nodes := make([]map[string]interface{}, len(views))
	for i, view := range views {
		nodes[i] = make(map[string]interface{})
		nodes[i]["id"] = view.ID
		nodes[i]["hostname"] = view.Hostname
		nodes[i]["ip"] = view.IP
	}

	return nodes
}

func updateMKSClusterV1KubeVersion(ctx context.Context, d *schema.ResourceData, client *v1.ServiceClient) error {
	o, n := d.GetChange("kube_version")
	currentVersion := o.(string)
	desiredVersion := n.(string)

	log.Printf("[DEBUG] current kube version: %s", currentVersion)
	log.Printf("[DEBUG] desired kube version: %s", desiredVersion)

	// Compare current and desired minor versions.
	currentMinor, err := kubeVersionToMinor(currentVersion)
	if err != nil {
		return fmt.Errorf("error getting a minor part of the current version %s: %s", currentVersion, err)
	}
	desiredMinor, err := kubeVersionToMinor(desiredVersion)
	if err != nil {
		return fmt.Errorf("error getting a minor part of the desired version %s: %s", desiredVersion, err)
	}
	if desiredMinor != currentMinor {
		return fmt.Errorf("current minor version %s can't be upgraded to %s", currentMinor, desiredMinor)
	}

	// Compare current and desired patch versions.
	currentPatch, err := kubeVersionToPatch(currentVersion)
	if err != nil {
		return fmt.Errorf("error getting a patch part of the current version %s: %s", currentVersion, err)
	}
	desiredPatch, err := kubeVersionToPatch(desiredVersion)
	if err != nil {
		return fmt.Errorf("error getting a patch part of the desired version %s: %s", desiredVersion, err)
	}
	if desiredPatch < currentPatch {
		return fmt.Errorf("current patch version %s can't be downgraded to %s", currentVersion, desiredVersion)
	}

	// Get all supported Kubernetes versions.
	kubeVersions, _, err := kubeversion.List(ctx, client)
	if err != nil {
		return fmt.Errorf("error getting all supported Kubernetes versions: %s", err)
	}

	// Find the latest patch version corresponding to the current minor version.
	var latestVersion string
	for _, version := range kubeVersions {
		minor, err := kubeVersionToMinor(version.Version)
		if err != nil {
			return err
		}
		if minor == currentMinor {
			if latestVersion == "" {
				latestVersion = version.Version
			} else {
				latestVersion, err = compareTwoKubeVersionsByPatch(latestVersion, version.Version)
				if err != nil {
					return err
				}
			}
		}
	}

	log.Printf("[DEBUG] latest kube version: %s", latestVersion)

	if desiredVersion != latestVersion {
		return fmt.Errorf(
			"current version %s can't be upgraded to version %s, the latest available patch version is: %s",
			currentVersion, desiredVersion, latestVersion)
	}

	_, err = cluster.UpgradePatchVersion(ctx, client, d.Id())
	if err != nil {
		return fmt.Errorf("error updating patch version: %s", err)
	}

	log.Printf("[DEBUG] waiting for cluster %s to become 'ACTIVE'", d.Id())
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = waitForMKSClusterV1ActiveState(ctx, client, d.Id(), timeout)
	if err != nil {
		return errUpdatingObject(objectCluster, d.Id(), err)
	}

	return nil
}

// kubeVersionToMinor returns given Kubernetes version trimmed to minor.
func kubeVersionToMinor(kubeVersion string) (string, error) {
	// Trim version prefix if needed.
	kubeVersion = strings.TrimPrefix(kubeVersion, "v")

	kubeVersionParts := strings.Split(kubeVersion, ".")
	if len(kubeVersionParts) < 2 {
		return "", errKubeVersionIsInvalidFmt(kubeVersion, "expected to have major and minor version parts")
	}

	majorPart := kubeVersionParts[0]
	major, err := strconv.Atoi(majorPart)
	if err != nil {
		return "", errKubeVersionIsInvalidFmt(kubeVersion, "major part is not an integer number")
	}
	if major < 0 {
		return "", errKubeVersionIsInvalidFmt(kubeVersion, "major part is a negative number")
	}

	minorPart := kubeVersionParts[1]
	minor, err := strconv.Atoi(minorPart)
	if err != nil {
		return "", errKubeVersionIsInvalidFmt(kubeVersion, "minor part is not an integer number")
	}
	if minor < 0 {
		return "", errKubeVersionIsInvalidFmt(kubeVersion, "minor part is a negative number")
	}

	return strings.Join([]string{majorPart, minorPart}, "."), nil
}

// kubeVersionToPatch returns given Kubernetes version patch part.
func kubeVersionToPatch(kubeVersion string) (int, error) {
	// Trim version prefix if needed.
	kubeVersion = strings.TrimPrefix(kubeVersion, "v")

	kubeVersionParts := strings.Split(kubeVersion, ".")
	if len(kubeVersionParts) < 3 {
		return 0, errKubeVersionIsInvalidFmt(kubeVersion, "expected to have major, minor and patch version parts")
	}

	patchPart := kubeVersionParts[2]
	patch, err := strconv.Atoi(patchPart)
	if err != nil {
		return 0, errKubeVersionIsInvalidFmt(kubeVersion, "patch part is not an integer number")
	}
	if patch < 0 {
		return 0, errKubeVersionIsInvalidFmt(kubeVersion, "patch part is a negative number")
	}

	return patch, nil
}

// compareTwoKubeVersionsByPatch parses two Kubernetes versions, compares their patch versions and returns
// the latest version.
// It doesn't check minor version so it will give bad result in case of different minor versions.
func compareTwoKubeVersionsByPatch(a, b string) (string, error) {
	aPatch, err := kubeVersionToPatch(a)
	if err != nil {
		return "", fmt.Errorf("unable to compare kube versions: %s", err)
	}

	bPatch, err := kubeVersionToPatch(b)
	if err != nil {
		return "", fmt.Errorf("unable to compare kube versions: %s", err)
	}

	if aPatch > bPatch {
		return a, nil
	}

	return b, nil
}
