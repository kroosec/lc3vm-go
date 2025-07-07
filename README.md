# About:

This is an implementation of a Virtual Machine for the
[LC-3](https://en.wikipedia.org/wiki/Little_Computer_3) computer, written in
Go.

It was inspired by [Justin Meiners' and Ryan Pendleton's
article](https://justinmeiners.github.io/lc3-vm/index.html).

It can run implementations of programs such as:

- [2048](https://github.com/rpendleton/lc3-2048):
- [Rogue](https://github.com/justinmeiners/lc3-rogue):

# Getting Started

## Build

To build the LC-3 VM executable, run the following command from the project root:

```bash
go build -o lc3vm ./cmd/lc3vm
```

This will create an executable named `lc3vm` in the project root directory.

## Run

To run an LC-3 object file, use the `lc3vm` executable followed by the path to the object file:

```bash
./lc3vm <path_to_object_file>
```

For example, to run the `hello-world.obj` example:

```bash
./lc3vm testdata/hello-world.obj
```

## Test

To run the test suite for the LC-3 VM, execute the following command from the project root:

```bash
go test -v ./...
```

# References:
- [LC-3 Instruction Set Architecture [PDF]](https://justinmeiners.github.io/lc3-vm/supplies/lc3-isa.pdf)
- [LC-3 Simulator](https://wchargin.github.io/lc3web/)
