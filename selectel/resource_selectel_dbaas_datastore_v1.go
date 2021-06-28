package selectel

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/selectel/dbaas-go"
)

func resourceDBaaSDatastoreV1() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDBaaSDatastoreV1Create,
		ReadContext:   resourceDBaaSDatastoreV1Read,
		UpdateContext: resourceDBaaSDatastoreV1Update,
		DeleteContext: resourceDBaaSDatastoreV1Delete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceDBaaSDatastoreV1ImportState,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(60 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"project_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					ru1Region,
					ru2Region,
					ru3Region,
					ru7Region,
					ru8Region,
					ru9Region,
				}, false),
			},
			"subnet_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"flavor_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"flavor"},
			},
			"node_count": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"connections": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"flavor": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      false,
				ConflictsWith: []string{"flavor_id"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vcpus": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
						"ram": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
						"disk": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},
			"pooler": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"mode": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: false,
							ValidateFunc: validation.StringInSlice([]string{
								"session",
								"transaction",
								"statement",
							}, false),
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: false,
						},
					},
				},
			},
			"firewall": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ips": {
							Type:     schema.TypeList,
							Required: true,
							ForceNew: false,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
			},
			"restore": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"datastore_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
						"target_time": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: false,
						},
					},
				},
			},
			"config": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceDBaaSDatastoreV1Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dbaasClient, diagErr := getDBaaSClient(ctx, d, meta)
	if diagErr != nil {
		return diagErr
	}

	flavorID, flavorIDOk := d.GetOk("flavor_id")
	flavorRaw, flavorOk := d.GetOk("flavor")

	if flavorIDOk == flavorOk {
		return diag.FromErr(errors.New("either 'flavor' or 'flavor_id' must be provided"))
	}

	poolerSet := d.Get("pooler").(*schema.Set)
	pooler, err := resourceDBaaSDatastoreV1PoolerFromSet(poolerSet)
	if err != nil {
		return diag.FromErr(errParseDatastoreV1Pooler(err))
	}

	restoreSet := d.Get("restore").(*schema.Set)
	restore, err := resourceDBaaSDatastoreV1RestoreOptsFromSet(restoreSet)
	if err != nil {
		return diag.FromErr(errParseDatastoreV1Restore(err))
	}

	configMap := d.Get("config").(map[string]interface{})
	config := make(map[string]interface{})
	for paramName, paramValue := range configMap {
		paramValueStr := paramValue.(string)
		config[paramName] = convertFieldFromStringToType(paramValueStr)
	}

	datastoreCreateOpts := dbaas.DatastoreCreateOpts{
		Name:      d.Get("name").(string),
		TypeID:    d.Get("type_id").(string),
		SubnetID:  d.Get("subnet_id").(string),
		NodeCount: d.Get("node_count").(int),
		Pooler:    pooler,
		Restore:   restore,
		Config:    config,
	}

	if flavorOk {
		flavorSet := flavorRaw.(*schema.Set)
		flavor, err := resourceDBaaSDatastoreV1FlavorFromSet(flavorSet)
		if err != nil {
			return diag.FromErr(errParseDatastoreV1Flavor(err))
		}

		datastoreCreateOpts.Flavor = flavor
	}

	if flavorIDOk {
		datastoreCreateOpts.FlavorID = flavorID.(string)
	}

	log.Print(msgCreate(objectDatastore, datastoreCreateOpts))
	datastore, err := dbaasClient.CreateDatastore(ctx, datastoreCreateOpts)
	if err != nil {
		return diag.FromErr(errCreatingObject(objectDatastore, err))
	}

	log.Printf("[DEBUG] waiting for datastore %s to become 'ACTIVE'", datastore.ID)
	timeout := d.Timeout(schema.TimeoutCreate)
	err = waitForDBaaSDatastoreV1ActiveState(ctx, dbaasClient, datastore.ID, timeout)
	if err != nil {
		return diag.FromErr(errCreatingObject(objectDatastore, err))
	}

	d.SetId(datastore.ID)

	return resourceDBaaSDatastoreV1Read(ctx, d, meta)
}

func resourceDBaaSDatastoreV1Read(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dbaasClient, diagErr := getDBaaSClient(ctx, d, meta)
	if diagErr != nil {
		return diagErr
	}

	log.Print(msgGet(objectDatastore, d.Id()))
	datastore, err := dbaasClient.Datastore(ctx, d.Id())
	if err != nil {
		return diag.FromErr(errGettingObject(objectDatastore, d.Id(), err))
	}
	d.Set("name", datastore.Name)
	d.Set("status", datastore.Status)
	d.Set("project_id", datastore.ProjectID)
	d.Set("subnet_id", datastore.SubnetID)
	d.Set("type_id", datastore.TypeID)
	d.Set("node_count", datastore.NodeCount)
	d.Set("enabled", datastore.Enabled)
	d.Set("flavor_id", datastore.FlavorID)

	_, flavorIDOk := d.GetOk("flavor_id")
	if !flavorIDOk {
		flavor := resourceDBaaSDatastoreV1FlavorToSet(datastore.Flavor)
		if err := d.Set("flavor", flavor); err != nil {
			log.Print(errSettingComplexAttr("flavor", err))
		}
	}

	connectonMap := make(map[string]string)
	connectonMap["master"] = datastore.Connection.Master
	if err := d.Set("connections", connectonMap); err != nil {
		log.Print(errSettingComplexAttr("connections", err))
	}

	return nil
}

func resourceDBaaSDatastoreV1Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dbaasClient, diagErr := getDBaaSClient(ctx, d, meta)
	if diagErr != nil {
		return diagErr
	}

	if d.HasChange("name") {
		err := updateDatastoreName(ctx, d, dbaasClient)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("pooler") {
		err := updateDatastorePooler(ctx, d, dbaasClient)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("firewall") {
		err := updateDatastoreFirewall(ctx, d, dbaasClient)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("node_count") || d.HasChange("flavor") || d.HasChange("flavor_id") {
		err := resizeDatastore(ctx, d, dbaasClient)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if d.HasChange("config") {
		err := updateDatastoreConfig(ctx, d, dbaasClient)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceDBaaSDatastoreV1Read(ctx, d, meta)
}

func resourceDBaaSDatastoreV1Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	dbaasClient, diagErr := getDBaaSClient(ctx, d, meta)
	if diagErr != nil {
		return diagErr
	}

	log.Print(msgDelete(objectDatastore, d.Id()))
	err := dbaasClient.DeleteDatastore(ctx, d.Id())
	if err != nil {
		return diag.FromErr(errDeletingObject(objectDatastore, d.Id(), err))
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{strconv.Itoa(http.StatusOK)},
		Target:     []string{strconv.Itoa(http.StatusNotFound)},
		Refresh:    dbaasDatastoreV1DeleteStateRefreshFunc(ctx, dbaasClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] waiting for datastore %s to become deleted", d.Id())
	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error waiting for the datastore %s to become deleted: %s", d.Id(), err))
	}

	return nil
}

func resourceDBaaSDatastoreV1ImportState(_ context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)
	if config.ProjectID == "" {
		return nil, errors.New("SEL_PROJECT_ID must be set for the resource import")
	}
	if config.Region == "" {
		return nil, errors.New("SEL_REGION must be set for the resource import")
	}

	d.Set("project_id", config.ProjectID)
	d.Set("region", config.Region)

	return []*schema.ResourceData{d}, nil
}

func waitForDBaaSDatastoreV1ActiveState(
	ctx context.Context, client *dbaas.API, datastoreID string, timeout time.Duration) error {
	pending := []string{
		string(dbaas.StatusPendingCreate),
		string(dbaas.StatusPendingUpdate),
		string(dbaas.StatusResizing),
	}
	target := []string{
		string(dbaas.StatusActive),
	}

	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     target,
		Refresh:    dbaasDatastoreV1StateRefreshFunc(ctx, client, datastoreID),
		Timeout:    timeout,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"error waiting for the datastore %s to become 'ACTIVE': %s",
			datastoreID, err)
	}

	return nil
}

func dbaasDatastoreV1StateRefreshFunc(ctx context.Context, client *dbaas.API, datastoreID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		d, err := client.Datastore(ctx, datastoreID)
		if err != nil {
			return nil, "", err
		}

		return d, string(d.Status), nil
	}
}

func dbaasDatastoreV1DeleteStateRefreshFunc(ctx context.Context, client *dbaas.API, datastoreID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		d, err := client.Datastore(ctx, datastoreID)
		if err != nil {
			var dbaasError *dbaas.DBaaSAPIError
			if errors.As(err, &dbaasError) {
				return d, strconv.Itoa(dbaasError.StatusCode()), nil
			}
			return nil, "", err
		}

		return d, strconv.Itoa(200), err
	}
}

func resourceDBaaSDatastoreV1FlavorFromSet(flavorSet *schema.Set) (*dbaas.Flavor, error) {
	if flavorSet.Len() == 0 {
		return nil, nil
	}
	var resourceVcpusRaw, resourceRAMRaw, resourceDiskRaw interface{}
	var ok bool

	resourceFlavorMap := flavorSet.List()[0].(map[string]interface{})
	if resourceVcpusRaw, ok = resourceFlavorMap["vcpus"]; !ok {
		return &dbaas.Flavor{}, errors.New("flavor.vcpus value isn't provided")
	}
	if resourceRAMRaw, ok = resourceFlavorMap["ram"]; !ok {
		return &dbaas.Flavor{}, errors.New("flavor.ram value isn't provided")
	}
	if resourceDiskRaw, ok = resourceFlavorMap["disk"]; !ok {
		return &dbaas.Flavor{}, errors.New("flavor.disk value isn't provided")
	}

	resourceVcpus := resourceVcpusRaw.(int)
	resourceRAM := resourceRAMRaw.(int)
	resourceDisk := resourceDiskRaw.(int)

	flavor := &dbaas.Flavor{
		Vcpus: resourceVcpus,
		RAM:   resourceRAM,
		Disk:  resourceDisk,
	}

	return flavor, nil
}

func resourceDBaaSDatastoreV1FlavorToSet(flavor dbaas.Flavor) *schema.Set {
	flavorSet := &schema.Set{
		F: flavorHashSetFunc(),
	}

	flavorSet.Add(map[string]interface{}{
		"vcpus": flavor.Vcpus,
		"ram":   flavor.RAM,
		"disk":  flavor.Disk,
	})

	return flavorSet
}

func flavorSchema() *schema.Resource {
	return resourceDBaaSDatastoreV1().Schema["flavor"].Elem.(*schema.Resource)
}

func flavorHashSetFunc() schema.SchemaSetFunc {
	return schema.HashResource(flavorSchema())
}

func resourceDBaaSDatastoreV1PoolerFromSet(poolerSet *schema.Set) (*dbaas.Pooler, error) {
	if poolerSet.Len() == 0 {
		return nil, nil
	}
	var resourceModeRaw, resourceSizeRaw interface{}
	var ok bool

	resourcePoolerMap := poolerSet.List()[0].(map[string]interface{})
	if resourceModeRaw, ok = resourcePoolerMap["mode"]; !ok {
		return &dbaas.Pooler{}, errors.New("pooler.mode value isn't provided")
	}
	if resourceSizeRaw, ok = resourcePoolerMap["size"]; !ok {
		return &dbaas.Pooler{}, errors.New("pooler.size value isn't provided")
	}

	resourceMode := resourceModeRaw.(string)
	resourceSize := resourceSizeRaw.(int)

	pooler := &dbaas.Pooler{
		Mode: resourceMode,
		Size: resourceSize,
	}

	return pooler, nil
}

func resourceDBaaSDatastoreV1PoolerOptsFromSet(poolerSet *schema.Set) (dbaas.DatastorePoolerOpts, error) {
	if poolerSet.Len() == 0 {
		return dbaas.DatastorePoolerOpts{}, nil
	}
	var resourceModeRaw, resourceSizeRaw interface{}
	var ok bool

	resourcePoolerMap := poolerSet.List()[0].(map[string]interface{})
	if resourceModeRaw, ok = resourcePoolerMap["mode"]; !ok {
		return dbaas.DatastorePoolerOpts{}, errors.New("pooler.mode value isn't provided")
	}
	if resourceSizeRaw, ok = resourcePoolerMap["size"]; !ok {
		return dbaas.DatastorePoolerOpts{}, errors.New("pooler.size value isn't provided")
	}

	resourceMode := resourceModeRaw.(string)
	resourceSize := resourceSizeRaw.(int)

	pooler := dbaas.DatastorePoolerOpts{
		Mode: resourceMode,
		Size: resourceSize,
	}

	return pooler, nil
}

func resourceDBaaSDatastoreV1FirewallOptsFromSet(firewallSet *schema.Set) (dbaas.DatastoreFirewallOpts, error) {
	if firewallSet.Len() == 0 {
		return dbaas.DatastoreFirewallOpts{IPs: []string{}}, nil
	}

	var resourceIPsRaw interface{}
	var ok bool

	resourceFirewallRaw := firewallSet.List()[0].(map[string]interface{})
	if resourceIPsRaw, ok = resourceFirewallRaw["ips"]; !ok {
		return dbaas.DatastoreFirewallOpts{}, errors.New("firewall.ips value isn't provided")
	}
	resourceIPRaw := resourceIPsRaw.([]interface{})
	var firewall dbaas.DatastoreFirewallOpts
	for _, ip := range resourceIPRaw {
		firewall.IPs = append(firewall.IPs, ip.(string))
	}

	return firewall, nil
}

func resourceDBaaSDatastoreV1RestoreOptsFromSet(restoreSet *schema.Set) (*dbaas.Restore, error) {
	if restoreSet.Len() == 0 {
		return nil, nil
	}
	var resourceDatastoreIDRaw, resourceTargetTimeRaw interface{}
	var ok bool

	resourceRestoreMap := restoreSet.List()[0].(map[string]interface{})
	if resourceDatastoreIDRaw, ok = resourceRestoreMap["datastore_id"]; !ok {
		return &dbaas.Restore{}, errors.New("restore.datastore_id value isn't provided")
	}
	if resourceTargetTimeRaw, ok = resourceRestoreMap["target_time"]; !ok {
		return &dbaas.Restore{}, errors.New("restore.target_time value isn't provided")
	}

	resourceDatastoreID := resourceDatastoreIDRaw.(string)
	resourceTargetTime := resourceTargetTimeRaw.(string)

	restore := &dbaas.Restore{
		DatastoreID: resourceDatastoreID,
		TargetTime:  resourceTargetTime,
	}

	return restore, nil
}

func convertFieldFromStringToType(fieldValue string) interface{} {
	if val, err := strconv.Atoi(fieldValue); err == nil {
		return val
	} else if val, err := strconv.ParseFloat(fieldValue, 64); err == nil {
		return val
	} else if val, err := strconv.ParseFloat(fieldValue, 32); err == nil {
		return val
	} else if val, err := strconv.ParseBool(fieldValue); err == nil {
		return val
	} else {
		return fieldValue
	}
}

func updateDatastoreName(ctx context.Context, d *schema.ResourceData, client *dbaas.API) error {
	var updateOpts dbaas.DatastoreUpdateOpts
	updateOpts.Name = d.Get("name").(string)

	log.Print(msgUpdate(objectDatastore, d.Id(), updateOpts))
	_, err := client.UpdateDatastore(ctx, d.Id(), updateOpts)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	log.Printf("[DEBUG] waiting for datastore %s to become 'ACTIVE'", d.Id())
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = waitForDBaaSDatastoreV1ActiveState(ctx, client, d.Id(), timeout)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	return nil
}

func updateDatastorePooler(ctx context.Context, d *schema.ResourceData, client *dbaas.API) error {
	poolerSet := d.Get("pooler").(*schema.Set)
	poolerOpts, err := resourceDBaaSDatastoreV1PoolerOptsFromSet(poolerSet)
	if err != nil {
		return errParseDatastoreV1Pooler(err)
	}

	log.Print(msgUpdate(objectDatastore, d.Id(), poolerOpts))
	_, err = client.PoolerDatastore(ctx, d.Id(), poolerOpts)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	log.Printf("[DEBUG] waiting for datastore %s to become 'ACTIVE'", d.Id())
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = waitForDBaaSDatastoreV1ActiveState(ctx, client, d.Id(), timeout)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	return nil
}

func updateDatastoreFirewall(ctx context.Context, d *schema.ResourceData, client *dbaas.API) error {
	firewallSet := d.Get("firewall").(*schema.Set)
	firewallOpts, err := resourceDBaaSDatastoreV1FirewallOptsFromSet(firewallSet)
	if err != nil {
		return errParseDatastoreV1Firewall(err)
	}

	log.Print(msgUpdate(objectDatastore, d.Id(), firewallOpts))
	_, err = client.FirewallDatastore(ctx, d.Id(), firewallOpts)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	log.Printf("[DEBUG] waiting for datastore %s to become 'ACTIVE'", d.Id())
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = waitForDBaaSDatastoreV1ActiveState(ctx, client, d.Id(), timeout)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	return nil
}

func updateDatastoreConfig(ctx context.Context, d *schema.ResourceData, client *dbaas.API) error {
	var configOpts dbaas.DatastoreConfigOpts
	datastore, err := client.Datastore(ctx, d.Id())
	if err != nil {
		return err
	}
	configMap := d.Get("config").(map[string]interface{})
	config := make(map[string]interface{})
	for paramName, paramValue := range configMap {
		paramValueStr := paramValue.(string)
		config[paramName] = convertFieldFromStringToType(paramValueStr)
	}

	for param := range datastore.Config {
		if _, ok := config[param]; !ok {
			config[param] = nil
		}
	}

	configOpts.Config = config

	log.Print(msgUpdate(objectDatastore, d.Id(), configOpts))
	_, err = client.ConfigDatastore(ctx, d.Id(), configOpts)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	log.Printf("[DEBUG] waiting for datastore %s to become 'ACTIVE'", d.Id())
	timeout := d.Timeout(schema.TimeoutUpdate)
	err = waitForDBaaSDatastoreV1ActiveState(ctx, client, d.Id(), timeout)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	return nil
}

func resizeDatastore(ctx context.Context, d *schema.ResourceData, client *dbaas.API) error {
	var resizeOpts dbaas.DatastoreResizeOpts
	nodeCount := d.Get("node_count").(int)
	resizeOpts.NodeCount = nodeCount

	flavorID := d.Get("flavor_id")
	flavorRaw := d.Get("flavor")

	flavorSet := flavorRaw.(*schema.Set)
	flavor, err := resourceDBaaSDatastoreV1FlavorFromSet(flavorSet)
	if err != nil {
		return errParseDatastoreV1Resize(err)
	}

	resizeOpts.Flavor = flavor
	resizeOpts.FlavorID = flavorID.(string)

	log.Print(msgUpdate(objectDatastore, d.Id(), resizeOpts))
	_, err = client.ResizeDatastore(ctx, d.Id(), resizeOpts)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	log.Printf("[DEBUG] waiting for datastore %s to become 'ACTIVE'", d.Id())
	timeout := d.Timeout(schema.TimeoutCreate)
	err = waitForDBaaSDatastoreV1ActiveState(ctx, client, d.Id(), timeout)
	if err != nil {
		return errUpdatingObject(objectDatastore, d.Id(), err)
	}

	return nil
}
