# Supersand

A lightweight, custom Linux container runtime built from scratch in Go. 

Supersand bypasses high-level container runtimes like `runc` or `containerd`, interfacing directly with Linux kernel primitives—namespaces, `pivot_root`, cgroups v2. It incorporates an integrated warm-pool scheduler to ensure sandboxes are pre-warmed and ready before execution requests arrive.

>[!NOTE]
>Supersand is an early-stage, actively evolving systems project with sharp edges, packages undergoing rewrites, and experimental components.

## Core Capabilities

* **Serverless Architecture & Security:** Designed to give users serverless execution with robust container security and strong Linux kernel isolation.
* **Optimized Response Times:** Prioritizes first-response time and minimal cold-start latency through pre-configured container setups powered by a warm worker pool.
* **Low-Level Isolation:** Spawns containers by re-executing binaries into isolated UTS, network, IPC, PID, and mount namespaces with custom rootfs construction and BusyBox layering.
* **PTY-Driven Execution & Resource Control:** Allocates pseudo-terminals (PTYs) with automated sentinel filtering alongside strict CPU, memory, and PID enforcement via cgroups v2.
* **Networking & APIs:** Manages container gRPC control planes.

## Roadmap

* **Async Worker Pool:** Implement an epoll-based worker pool to replace the current dispatcher stub, optimizing warm-pool task handoff and reducing per-container goroutine overhead.
* **File-Backed LRU Cache:** Implement disk spilling for evicted session and process data to optimize memory utilization as the warm pool scales.
* **Container Pre-Forking:** Evaluate Zygote-style pre-forking mechanisms (potentially utilizing Rust components) to achieve lower-latency, more isolated container startups than current PTY-based launches.
* **Network Egress Routing:** Integrate `nftables` rules for container network traffic forwarding and egress control.
* **Service Completion:** Finalize gRPC service implementations and fully integrate Biscuit token authentication into the request middleware pipeline.



