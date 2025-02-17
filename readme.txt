env variables needed for compile:


set CGO_LDFLAGS=-LC:\sdl2\lib\x64 -lSDL2
set CGO_CFLAGS=-IC:\sdl2\include

set CGO_LDFLAGS=-LC:\SDL2_gfx\lib -lSDL2_gfx
set CGO_CFLAGS=-IC:\SDL2_gfx\include


this is for vcpkg:
set CGO_CFLAGS=-IC:\Users\Zabicka\vcpkg\installed\x64-windows\include
set CGO_LDFLAGS=-LC:\Users\Zabicka\vcpkg\installed\x64-windows\lib -lSDL2 -lSDL2_gfx -lSDL2_ttf

To run the server:
go run main.go -mode=server -address=localhost:9000


To run the client:
go run main.go -mode=client -address=localhost:9000



