package main

const syscall_config string = `str.go
syscall.go
syscall_386.go
syscall_linux.go
syscall_linux_386.go
zerrors_linux_386.go
zsyscall_linux_386.go
zsysnum_linux_386.go
ztypes_linux_386.go
syscall_unix.go
exec_unix.go

`

const runtime_config string = `debug.go
error.go
extern.go
sig.go
softfloat64.go
type.go
version.go
version_linux.go
version_386.go
chan_defs.go
hashmap_defs.go
iface_defs.go
malloc_defs.go
mheapmap32_defs.go
runtime_defs.go
linux/runtime_defs.go

`

const os_config string = `dir_linux.go
error.go
env.go
exec.go
file.go
getwd.go
path.go
proc.go
stat_linux.go
time.go
types.go
env_unix.go
file_unix.go
sys_linux.go

`

const goref_config_stub string = `.packages
.externPackages
.specialPackages
syscall
os
runtime`
