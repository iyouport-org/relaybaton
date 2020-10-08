package memsocket

const (
	Network = "memsocket"
)

type Addr struct {
}

// name of the network
func (addr Addr) Network() string {
	return Network
}

// string form of address
func (addr Addr) String() string {
	return Network
}
