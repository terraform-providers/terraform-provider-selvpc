package dbaas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Instances represents datastore's instances.
type Instances struct {
	IP       string `json:"ip"`
	Role     string `json:"role"`
	Status   Status `json:"status"`
	Hostname string `json:"hostname"`
}

// Connection represents datastore's connection.
type Connection struct {
	Master string `json:"MASTER"`
}

// Flavor represents datastore's flavor.
type Flavor struct {
	Vcpus int `json:"vcpus"`
	RAM   int `json:"ram"`
	Disk  int `json:"disk"`
}

// Restore represents restore parameters for datastore.
type Restore struct {
	DatastoreID string `json:"datastore_id,omitempty"`
	TargetTime  string `json:"target_time,omitempty"`
}

// Pooler represents pooler parameters for datastore.
type Pooler struct {
	Mode string `json:"mode,omitempty"`
	Size int    `json:"size,omitempty"`
}

// Firewall represents firewall rules parameters for datastore.
type Firewall struct {
	IP string `json:"ip"`
}

// Datastore is the API response for the datastores.
type Datastore struct {
	ID         string                 `json:"id"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
	ProjectID  string                 `json:"project_id"`
	Name       string                 `json:"name"`
	TypeID     string                 `json:"type_id"`
	SubnetID   string                 `json:"subnet_id"`
	FlavorID   string                 `json:"flavor_id"`
	Status     Status                 `json:"status"`
	Connection Connection             `json:"connection"`
	Firewall   []Firewall             `json:"firewall"`
	Instances  []Instances            `json:"instances"`
	Config     map[string]interface{} `json:"config"`
	Pooler     Pooler                 `json:"pooler"`
	Flavor     Flavor                 `json:"flavor"`
	NodeCount  int                    `json:"node_count"`
	Enabled    bool                   `json:"enabled"`
}

// DatastoreCreateOpts represents options for the datastore Create request.
type DatastoreCreateOpts struct {
	Flavor    *Flavor                `json:"flavor,omitempty"`
	Restore   *Restore               `json:"restore,omitempty"`
	Pooler    *Pooler                `json:"pooler,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Name      string                 `json:"name"`
	TypeID    string                 `json:"type_id"`
	SubnetID  string                 `json:"subnet_id"`
	FlavorID  string                 `json:"flavor_id,omitempty"`
	NodeCount int                    `json:"node_count"`
}

// DatastoreUpdateOpts represents options for the datastore Update request.
type DatastoreUpdateOpts struct {
	Name string `json:"name"`
}

// DatastoreResizeOpts represents options for the datastore Resize request.
type DatastoreResizeOpts struct {
	Flavor    *Flavor `json:"flavor,omitempty"`
	FlavorID  string  `json:"flavor_id,omitempty"`
	NodeCount int     `json:"node_count,omitempty"`
}

// DatastorePoolerOpts represents options for the datastore's pooler Update request.
type DatastorePoolerOpts struct {
	Mode string `json:"mode,omitempty"`
	Size int    `json:"size,omitempty"`
}

// DatastoreFirewallOpts represents options for the datastore's firewall rules Ureate request.
type DatastoreFirewallOpts struct {
	IPs []string `json:"ips"`
}

type DatastoreConfigOpts struct {
	Config map[string]interface{} `json:"config"`
}

// DatastoreQueryParams represents available query parameters for datastore.
type DatastoreQueryParams struct {
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Status    Status `json:"status,omitempty"`
	Enabled   string `json:"enabled,omitempty"`
	TypeID    string `json:"type_id,omitempty"`
	SubnetID  string `json:"subnet_id,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

// Datastores returns all datastores.
func (api *API) Datastores(ctx context.Context, params *DatastoreQueryParams) ([]Datastore, error) {
	uri, err := setQueryParams("/datastores", params)
	if err != nil {
		return []Datastore{}, err
	}

	resp, err := api.makeRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return []Datastore{}, err
	}

	var result struct {
		Datastores []Datastore `json:"datastores"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return []Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastores, nil
}

// Datastore returns a datastore based on the ID.
func (api *API) Datastore(ctx context.Context, datastoreID string) (Datastore, error) {
	uri := fmt.Sprintf("/datastores/%s", datastoreID)

	resp, err := api.makeRequest(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}

// CreateDatastore creates a new datastore.
func (api *API) CreateDatastore(ctx context.Context, opts DatastoreCreateOpts) (Datastore, error) {
	uri := "/datastores"
	createDatastoreOpts := struct {
		Datastore DatastoreCreateOpts `json:"datastore"`
	}{
		Datastore: opts,
	}
	requestBody, err := json.Marshal(createDatastoreOpts)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error marshalling params to JSON, %w", err)
	}

	resp, err := api.makeRequest(ctx, http.MethodPost, uri, requestBody)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}

// UpdateDatastore updates an existing datastore.
func (api *API) UpdateDatastore(ctx context.Context, datastoreID string, opts DatastoreUpdateOpts) (Datastore, error) {
	uri := fmt.Sprintf("/datastores/%s", datastoreID)
	updateDatastoreOpts := struct {
		Datastore DatastoreUpdateOpts `json:"datastore"`
	}{
		Datastore: opts,
	}
	requestBody, err := json.Marshal(updateDatastoreOpts)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error marshalling params to JSON, %w", err)
	}

	resp, err := api.makeRequest(ctx, http.MethodPut, uri, requestBody)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}

// DeleteDatastore deletes an existing datastore.
func (api *API) DeleteDatastore(ctx context.Context, datastoreID string) error {
	uri := fmt.Sprintf("/datastores/%s", datastoreID)

	_, err := api.makeRequest(ctx, http.MethodDelete, uri, nil)
	if err != nil {
		return err
	}

	return nil
}

// ResizeDatastore resizes an existing datastore.
func (api *API) ResizeDatastore(ctx context.Context, datastoreID string, opts DatastoreResizeOpts) (Datastore, error) {
	uri := fmt.Sprintf("/datastores/%s/resize", datastoreID)
	resizeDatastoreOpts := struct {
		Datastore DatastoreResizeOpts `json:"resize"`
	}{
		Datastore: opts,
	}
	requestBody, err := json.Marshal(resizeDatastoreOpts)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error marshalling params to JSON, %w", err)
	}

	resp, err := api.makeRequest(ctx, http.MethodPost, uri, requestBody)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}

// PoolerDatastore updates pooler parameters of an existing datastore.
func (api *API) PoolerDatastore(ctx context.Context, datastoreID string, opts DatastorePoolerOpts) (Datastore, error) {
	uri := fmt.Sprintf("/datastores/%s/pooler", datastoreID)
	poolerDatastoreOpts := struct {
		Datastore DatastorePoolerOpts `json:"pooler"`
	}{
		Datastore: opts,
	}
	requestBody, err := json.Marshal(poolerDatastoreOpts)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error marshalling params to JSON, %w", err)
	}

	resp, err := api.makeRequest(ctx, http.MethodPut, uri, requestBody)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}

// FirewallDatastore updates firewall rules of an existing datastore.
func (api *API) FirewallDatastore(ctx context.Context, datastoreID string, opts DatastoreFirewallOpts) (Datastore, error) { //nolint
	uri := fmt.Sprintf("/datastores/%s/firewall", datastoreID)
	firewallDatastoreOpts := struct {
		Datastore DatastoreFirewallOpts `json:"firewall"`
	}{
		Datastore: opts,
	}
	requestBody, err := json.Marshal(firewallDatastoreOpts)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error marshalling params to JSON, %w", err)
	}

	resp, err := api.makeRequest(ctx, http.MethodPut, uri, requestBody)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}

// ConfigDatastore updates firewall rules of an existing datastore.
func (api *API) ConfigDatastore(ctx context.Context, datastoreID string, opts DatastoreConfigOpts) (Datastore, error) { //nolint
	uri := fmt.Sprintf("/datastores/%s/config", datastoreID)
	requestBody, err := json.Marshal(opts)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error marshalling params to JSON, %w", err)
	}

	resp, err := api.makeRequest(ctx, http.MethodPut, uri, requestBody)
	if err != nil {
		return Datastore{}, err
	}

	var result struct {
		Datastore Datastore `json:"datastore"`
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		return Datastore{}, fmt.Errorf("Error during Unmarshal, %w", err)
	}

	return result.Datastore, nil
}
