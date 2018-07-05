package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/scaleway/scaleway-cli/pkg/api"
)

// DynamicInventory define ansible inventory base structure
type DynamicInventory struct {
	Metadata MetaHostvars
	Groups   map[string]*Group
}

// MetaHostvars contain metadata hostvars
type MetaHostvars struct {
	Hosts map[string]map[string]map[string]string `json:"hostvars"`
}

// Group define ansible group structure
type Group struct {
	Hosts    []string `json:"hosts,omitempty"`
	Children []string `json:"children,omitempty"`
}

// MarshalJSON implements json.Marshaler
func (a *DynamicInventory) MarshalJSON() ([]byte, error) {
	// Define a map that we will encode at the end of the function.
	// Add all the _meta key then add all groups

	doc := make(map[string]interface{})

	doc["_meta"] = a.Metadata

	for k, v := range a.Groups {
		doc[k] = *v
	}

	return json.Marshal(doc)
}

func main() {

	var dynamicInventory DynamicInventory
	orgToken := os.Getenv("SCALEWAY_ORGANIZATION")
	token := os.Getenv("SCALEWAY_TOKEN")

	if strings.TrimSpace(orgToken) == "" || strings.TrimSpace(token) == "" {
		panic("required environmental variables are not set")
	}

	getServers(&dynamicInventory, token, orgToken)

	body, err := json.Marshal(&dynamicInventory)
	if err != nil {
		panic("failed to marshal the dynamic inventory")
	}

	fmt.Println(string(body))
}

func getServers(d *DynamicInventory, token, orgToken string) {
	disabledLoggerFunc := func(a *api.ScalewayAPI) {
		a.Logger = api.NewDisableLogger()
	}

	api, err := api.NewScalewayAPI(orgToken, token, "Scaleway Dynamic Inventory",
		"", disabledLoggerFunc)
	if err != nil {
		panic(fmt.Sprintf("failed to create API instance: %s", err))
	}

	servers, err := api.GetServers(true, 0)
	if err != nil {
		panic(fmt.Sprintf("failed to get servers: %s", err))
	}

	// Create Metadata
	if d.Metadata.Hosts == nil {
		d.Metadata.Hosts = make(map[string]map[string]map[string]string, 0)
	}

	//Create Group
	if d.Groups == nil {
		d.Groups = make(map[string]*Group, 0)
	}

	count := 10
	for _, server := range *servers {
		// Add hostvars
		if _, ok := d.Metadata.Hosts[server.PrivateIP]; !ok {
			d.Metadata.Hosts[server.PrivateIP] = make(map[string]map[string]string, 0)
			d.Metadata.Hosts[server.PrivateIP]["scaleway"] = make(map[string]string, 0)
		}
		// Internal VPN IP
		count++
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["vpn_ip"] = "192.168.0." + strconv.Itoa(count)
		// Passthrough server information
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["arch"] = server.Arch
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["bootscript"] = server.Bootscript.Identifier
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["commercialtype"] = server.CommercialType
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["creationdate"] = server.CreationDate
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["privatedns"] = server.DNSPrivate
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["publicdns"] = server.DNSPublic
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["hostname"] = server.Hostname
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["identifier"] = server.Identifier
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["image"] = server.Image.Name
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["name"] = server.Name
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["organization"] = server.Organization
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["privateip"] = server.PrivateIP
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["publicip"] = server.PublicAddress.IP
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["state"] = server.State
		tags, _ := json.Marshal(server.Tags)
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["tags"] = strings.Replace(string(tags), "\"", "", -1)
		volumes, _ := json.Marshal(server.Volumes)
		d.Metadata.Hosts[server.PrivateIP]["scaleway"]["volumes"] = strings.Replace(string(volumes), "\"", "", -1)

		// Add server inside ansible groups
		for _, tag := range server.Tags {
			if _, ok := d.Groups[tag]; !ok {
				d.Groups[tag] = new(Group)
				d.Groups[tag].Hosts = make([]string, 0)
				d.Groups[tag].Children = make([]string, 0)
			}

			d.Groups[tag].Hosts = append(d.Groups[tag].Hosts, server.PrivateIP)

			// if server.PublicAddress.IP != "" {
			// 	dict[tag] = append(dict[tag], server.PublicAddress.IP)
			// } else {
			// 	dict[tag] = append(dict[tag], server.PrivateIP)
			// }
		}
	}
}
