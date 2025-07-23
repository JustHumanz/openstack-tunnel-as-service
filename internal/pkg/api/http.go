package api

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
	// Convert []tunnel.VmTunnel to []db.VmTunnelJson for consistent JSON output
	var tunnelsJson []db.VmTunnelJson
	for _, t := range tunnels {
		tunnelsJson = append(tunnelsJson, db.VmTunnelJson{
			VMname: t.VMname,
			VMID:   t.VMID,
			VMSvc:  t.VMSvc,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tunnelsJson)
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
		return
	}
	// Convert []tunnel.VmTunnel to []db.VmTunnelJson for consistent JSON output
	for index, t := range tunnels {
		if t.VMID == vmID {
			TunnelVMs := i.TunnelVMs
			Prov := i.TunnelVMs.TunProvider
			NG := Prov.NG
			CF := Prov.CF

			switch {
			case NG.Active:
				t.StopNgrok(NG, "")
				return

			case CF.Active:
				err := t.StopCloudFlare(CF, "")
				if err != nil {
					Log.Error(err)
					http.Error(w, "Failed to stop Cloudflare tunnel", http.StatusInternalServerError)
					return
				}

			default:
				http.Error(w, "No active tunnel provider found", http.StatusInternalServerError)
				return
			}

			TunnelVMs.RemoveTunnelsByIndex(index)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Tunnel for VM %s deleted successfully", vmID),
		})
	}
}
