# GoRefactor

GoRefactor is a tool for refactoring go programs.

## Requirements

[Go language compiler](http://golang.org/doc/install.html) is required to build GoRefactor. This version of GoRefactor works with r58 release of Go compiler.

## Installation

Once you have the go compiler installed, you can proceed to installing GoRefactor. Simply run the `scripts/build` script. It installs GoRefactor via [gomake](http://golang.org/cmd/gomake/).

## Preparing code for GoRefactor

To allow GoRefactor work with your source code project, you need to prepare it in a few simple steps:

1.    Project structure should be canonical. That means, every package should be in it's own folder, named exactly as the package. Nested packages are allowed. All the packages must be contained in single parent folder (let's name it **source folder**).
You can look at the correct project structure in the GoRefactor source (the `src` folder).

2.   You must install your packages via gomake install (look [here](http://golang.org/cmd/gomake/) and [here](http://golang.org/doc/code.html)). If you don't already do this, you should really check it out, it's really convenient.

    *NOTE: GoRefactor source is an example of a GoRefactor project, you can explore it, if you have troubles.*

3.    **Source folder** must contain `goref.cfg` file. It's a text file, listing all the packages of your project in a special way. It has 3 sections:

    * `.packages` section. Each line of it describes a package as a pair {relative_path, go_source_path}. *relative_path* is the path to the package folder, relative to the **source folder**. *go_source_path* is the place in the go source folder (usually, `~/go/src`), where you install your package. You can find *go_source_path* in the `TARG` variable of `Makefile` for your package.

        If package is not installed at 'go/src' (executable package), place a `_` symbol instead of *go_source_path*.

    * `.externPackages` section. This section is optional. Here you can specify external dependencies (out of **source folder**). Packages are listed in the same way, the only difference is that instead of *relative_path* you should use the absolute path of the package.

    * `.specialPackages` section. This section contains a list of architecture-dependent packages. For now, there're 3 architecture-dependent packages in go sources: syscall, os, and runtime. The all must be listed there.

    You can get a working `goref.cfg` file from this repository, and edit it.

4.    For each architecture-dependent package **source folder** must contain a `.cfg` file, named as the package. For now, you can just copy `syscall.cfg`, `os.cfg` and `runtime.cfg` from this repository to your project, and copy the `.specialPackages` section of `goref.cfg`.

    *NOTE: all this additional config stuff was added during development, because of regular changes in go compiler source.*

**Again, you can use `src` folder of this repository to set up your own GoRefactor project**

## Usage

GoRefactor can perform 6 actions. All of them listed below. **For now, all paths in command line parameters must be absolute**.

Rename

    usage: goref ren <filename> <line> <column> <new name>

Extract Method

    usage: goref exm <filename> <line> <column> <end line> <end column> <new name> [<recvLine> <recvColumn>]

Inline Method

    usage: goref inm <filename> <line> <column> <end line> <end column>

Implement Interface

    usage: goref imi [-p] <filename> <line> <column> <type line> <type column>
    -p: implement interface for pointerType

Extract Interface

    usage: goref exi <filename> <line> <column> <interface name>

Sort declarations within file

    usage: goref sort [-t|-v] [-i] <filename> [<order>]

    -t:      group methods by reciever type. Methods will be sorted alphabetically by name within group.
    -v:      group methods by visibility. Methods will be sorted alphabetically by name within group.
    -i:      sort imports alphabetically.
    <order>: defines custom order of groups of declarations.
    Default order string is 'cvtmf' which means 'constants, variables, types, methods, functions'
    Custom order string must contain at least one character from default order string.
    If it's length is less than the length of default order string, other entries will be added in the default order.
    Leave out order parameter to use default order.