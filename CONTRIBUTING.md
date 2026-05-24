# contributing to apnoptes

you want to add features? cool. but don't make a mess.

we use **hexagonal architecture** (ports and adapters) here. if you don't know what that is, read up before you submit a PR. 

## the golden rules

1. **core is sacred:** the core proxy engine does not know about the TUI. it does not know about SQLite. it only knows about intercepting traffic and pushing data to channels.
2. **strict decoupling:** 
   - **adapters:** your SQLite storage logic and Bubble Tea presentation states are adapters. 
   - **ports:** define interfaces in the core. let the adapters implement them.
3. **zero memory allocation:** stream those HTTP request/response body data using `io.Copy` buffers. do not read massive binary blobs (like 2GB zip files) into heap memory. 
4. **non-blocking logging:** disk writes never happen on the active socket's network goroutine loop. throw it into the ring-buffer channel and let the async worker pool handle it.

## how to structure your code

keep the network layer, storage layer, and presentation layer separated. if i see UI logic inside the HTTP mitm layer, i will close your PR.

## running tests

if you touch the async pipeline, make sure you run parallel benchmark scripts to validate zero network request thread blockage during extreme write operations.

stay fast. go brrr.
