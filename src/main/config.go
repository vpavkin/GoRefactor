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
debug.go
error.go
extern.go
sig.go
mem.go
softfloat64.go
type.go
version.go
version_linux.go
version_386.go
runtime_defs.go

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
dir_unix.go
error_posix.go
env_unix.go
file_posix.go
file_unix.go
sys_linux.go
exec_posix.go
exec_unix.go

`

const goref_config_stub string = `.packages
.externPackages
.specialPackages
syscall
os
runtime`
