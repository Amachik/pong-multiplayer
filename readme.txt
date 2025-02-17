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



In your main application (main.go) you would continue to use the client’s Send method and the server’s Broadcast method exactly as before—but now the messages are encoded in binary and sent over UDP.
This design avoids the overhead and head-of-line blocking of TCP/JSON, and (with appropriate further enhancements such as packet sequencing, client‐side prediction/interpolation, and possibly switching to a more complete UDP reliability layer) can help you achieve a production‐ready online experience.