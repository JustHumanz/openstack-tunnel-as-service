package api

/// TODO: Rewrite using gorilla/mux or similar for better path param handling

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

var (
	Log = pkg.Log // Use the log from pkg/log.go
)

type APIops struct {
	ListenPort int
	TunnelVMs  *tunnel.TunnelData
}

func (i *APIops) StartAPI() {
	Log.Info("Starting tunnel as service API")

	// Start API
	http.HandleFunc("/api/tunnelvms", i.ListVmTunnelsHandler)
	http.HandleFunc("/api/{VMID}", i.DeleteVmTunnelsHandler)
	Log.Infof("API server listening on :%d", i.ListenPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", i.ListenPort), nil); err != nil {
		Log.Fatal(err)
	}

}

func (i *APIops) ListVmTunnelsHandler(w http.ResponseWriter, r *http.Request) {
	tunnels, err := db.LoadTunnels()
	if err != nil {
		Log.Error(err)
		http.Error(w, "Failed to load tunnels", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tunnels)
}

func (i *APIops) DeleteVmTunnelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vmID := r.PathValue("VMID")
	tunnels, err := db.LoadTunnels()
	if err != nil {
		Log.Error(err)
		http.Error(w, "Failed to load tunnels", http.StatusInternalServerError)
		return
	}

	tmpTunnel := []tunnel.InstanceTunnel{}
	for _, t := range tunnels {
		if t.InstanceID == vmID {
			Prov := i.TunnelVMs.TunProvider

			switch {
			case Prov.CF.Active:
				err := t.DeleteCFTunnel(nil, &Prov)
				if err != nil {
					Log.Error(err)
				}
			case Prov.NG.Active:
				err := t.DeleteNGTunnel(nil, &Prov)
				if err != nil {
					Log.Error(err)
				}
			default:
				Log.Error("Invalid Provider")
			}
		} else {
			tmpTunnel = append(tmpTunnel, t)
		}
	}

	err = db.SaveTunnels(tmpTunnel)
	if err != nil {
		Log.Error(err)
		http.Error(w, "Failed to save tunnels", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": fmt.Sprintf("Tunnel for VM %s deleted successfully", vmID),
	})
}
