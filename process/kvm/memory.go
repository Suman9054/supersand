package kvm

// this is to stup the memory region for the kvm and setup shard memory for ipc
type UserMemoryRegion struct {
    Slot          uint32
    Flags         uint32
    GuestPhysAddr uint64
    MemorySize    uint64
    UserspaceAddr uint64
}
