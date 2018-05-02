package selvpc

import "github.com/selectel/go-selvpcclient/selvpcclient/resell/v2/servers"

// serversMapsFromStructs converts the provided license.Servers to
// the slice of maps correspondingly to the resource's schema.
func serversMapsFromStructs(serverStructs []servers.Server) []map[string]interface{} {
	associatedServers := make([]map[string]interface{}, len(serverStructs))

	if len(serverStructs) != 0 {
		for i, server := range serverStructs {
			associatedServers[i] = map[string]interface{}{
				"id":     server.ID,
				"name":   server.Name,
				"status": server.Status,
			}
		}
	}

	return associatedServers
}