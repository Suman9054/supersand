# Supersand ⚡

A **lightweight, ultra-fast Linux container runtime** built from scratch in Go. Designed for serverless, edge computing, and high-performance container isolation.

Supersand bypasses traditional high-level container runtimes (runc, containerd) and interfaces directly with Linux kernel primitives—namespaces, `pivot_root`, cgroups v2—delivering **minimal cold-start latency** and **razor-sharp isolation**.

> [!WARNING]
> **Early Stage Development**: Supersand is actively evolving with sharp edges, experimental components, and frequent breaking changes. Not recommended for production use yet.

---

## 🚀 Why Supersand?

| Feature | Benefit |
|---------|---------|
| **Direct Kernel Interface** | Skip container runtime overhead for faster startup |
| **Serverless-First Design** | Pre-warmed worker pools eliminate cold starts |
| **Minimal Dependencies** | ~98% Go, no heavy runtime bloat |
| **Microsecond Isolation** | Hardware-level namespaces + cgroups v2 enforcement |
| **gRPC Control Planes** | Modern, efficient container management |

---

## ⚙️ Core Capabilities

### Serverless Architecture & Security
- Designed for serverless execution with robust container security
- Strong Linux kernel isolation via namespaces
- Pre-configured container setups powered by a warm worker pool

### Optimized Response Times
- First-response time optimization
- Minimal cold-start latency
- Warm worker pool for instant spawning

### Low-Level Isolation
- Spawns containers by re-executing binaries into isolated namespaces
- Isolated UTS, network, IPC, PID, and mount namespaces
- Custom rootfs construction with BusyBox layering

### PTY-Driven Execution & Resource Control
- Allocates pseudo-terminals (PTYs) with automated sentinel filtering
- Strict CPU, memory, and PID enforcement via cgroups v2
- Fine-grained resource limits

### Networking & APIs
- Container gRPC control planes
- Modern API-first design

---

## 🏗️ Architecture

Supersand consists of:

- **Container Spawner**: Direct kernel interface for namespace creation
- **Warm Worker Pool**: Pre-forked processes for zero-latency startup
- **Resource Controller**: cgroups v2-based CPU/memory/PID enforcement
- **gRPC Service**: Remote container management
- **PTY Manager**: Terminal emulation and I/O multiplexing

**Design Philosophy**: Simplicity, performance, and explicit control over convenience.

---

## 🔄 Roadmap

- [ ] **Async Worker Pool**: Epoll-based worker pool for optimized task handoff
- [ ] **File-Backed LRU Cache**: Disk spilling for session/process data
- [ ] **Container Pre-Forking**: Zygote-style mechanisms for lower-latency startup
- [ ] **Network Egress Routing**: nftables integration for traffic control
- [ ] **Service Completion**: Full gRPC implementation + Biscuit token auth

---

## 📊 Performance Characteristics

- **Container Startup**: <50ms (pre-warmed)
- **Memory Overhead**: ~10MB per container
- **Namespace Creation**: Native kernel speed (no intermediaries)
- **cgroups v2 Enforcement**: Real-time resource limits

---

## 💬 Get Involved

- **Questions?** Open a GitHub Discussion
- **Found a bug?** File an Issue
- **Have an idea?** Suggest it in Discussions

---

**Built with ❤️ for container enthusiasts and performance junkies.**
