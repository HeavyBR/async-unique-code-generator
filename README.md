# async-unique-code-generator
An async unique code generator written in pure GO that can generates millions of code in a couple of milliseconds and save to a file

The code generation process take into account that in a production environment you will use an async distributed way to save the codes and adds a couple of network latency simulation to be more similar to a production environment.

# Usage

`go run main.go -size 10 -quantity 10000000 -prefix TEST -output codes.txt`
