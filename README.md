# OpenStack Tunnel as a Service
A lightweight tunneling service for accessing private OpenStack clusters. This project allows you to expose internal services (such as SSH, APIs, etc.) securely via public tunnels using either ngrok or Cloudflare Tunnel.

## ✨ Features
- Expose private OpenStack services via public endpoints
- Supports ngrok and Cloudflare Tunnel as backends
- Easy environment variable configuration
- Headless operation, suitable for automation or CI/CD

## 📦 Requirements
- Access to a private OpenStack cluster
- A working installation of Go (for building the tool)
- Either:
  - A valid ngrok auth token, or
  - A valid Cloudflare API key

## ⚙️ Setup
1. Clone the Repository

```bash
git clone https://github.com/JustHumanz/openstack-tunnel-as-service.git
cd openstack-tunnel-as-service
```

2. Export Your Tunnel Backend Credentials

```bash
export NGROK_AUTHTOKEN=your-ngrok-authtoken
OR
export CLOUDFLARE_API_KEY=your-cloudflare-api-key
```

3. Build the Project

```bash
go build -o tunnel-service main.go
```

4. Run the Service

```bash
./tunnel-service 
#OR when using cloudflare
./tunnel-service -cf /usr/bin/cloudflared -domain kano2525.dev
```

5. Start Labeling VMS

```bash
openstack server set --property tunnel='ssh' cirros
# wait until service pickup vm metadata property
openstack server show cirros 
```
The scheduler will pickup the labels and automatically start a tunnel using your chosen backend.


## 🛠 Supported Tunnel Backends
| Backend    | Env Variable         | Notes                                    |
| ---------- | -------------------- | ---------------------------------------- |
| ngrok      | `NGROK_AUTHTOKEN`    | Requires ngrok account                   |
| Cloudflare | `CLOUDFLARE_API_KEY` | Requires active zone setup in Cloudflare |


## 🧪 Example Use Cases
- Allowing temporary SSH access to OpenStack vms

## 🚧 Roadmap / TODOs
- Add support for multiple tunnel instances
- Better error handling and retry mechanism

## 📄 License
IDK, MIT i guess