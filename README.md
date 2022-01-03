# Go EventEmitter Interface

This project intends to implements a JavaScript-like EventEmitter for Golang

# Implementations

## Types

### EventCallback

Just an alias to your callback function for **On** and **Once** event handlers

### EventCallable

Just a struct that hold your callback function and a flag indicate that it should be execute just
once or not

### EventHandler

This struct holds a list of a **EventCallable** with all handlers that should be execute for a specific
event

### EventEmitter

This is the main struct of this package, it implements functions for event handling like **On**,
**Once**, **PrependListener** and so on, and a function **Emit** for event issuing, just like
the same way that JavaScript's EventEmitter does


