package network

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

type Package struct {
	Option int
	Data   string
}

const (
	ENDBYTES   = "\000\005\007\001\001\007\005\000"
	WAITTIME   = 1 * time.Second
	DMAXSIZE   = (2 << 20)
	BUFFSIZE   = (4 << 10)
	WAKEUP_MSG = 6
)

type Listener = net.Listener
type Conn = net.Conn

// Listen starts a network listener on the specified address, and sets up a handler for incoming connections
func Listen(address string, handle func(Conn, *Package)) Listener {
	splited := strings.Split(address, ":")
	if len(splited) != 2 {
		return nil
	}
	listener, err := net.Listen("tcp", "0.0.0.0:"+splited[1])
	if err != nil {
		fmt.Println("Error while starting listener:", err)
		return nil
	}
	go serve(listener, handle)
	return listener
}

// serve starts the service and handles incoming connections
func serve(listener net.Listener, handler func(Conn, *Package)) {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error while accepting connection:", err)
			break
		}
		go handleConn(conn, handler)
	}
}

// Handle handles incoming requests based on option and responds using the provided handle function
func Handle(option int, conn Conn, pack *Package, handle func(*Package) string) bool {
	if pack.Option != option {
		return false
	}
	response := SerializePackage(&Package{
		Option: option,
		Data:   handle(pack),
	})
	_, err := conn.Write([]byte(response + ENDBYTES))
	if err != nil {
		fmt.Println("Error while sending response:", err)
		return false
	}
	return true
}

// handleConn reads the package from the connection and uses the provided handler to handle it
func handleConn(conn net.Conn, handle func(Conn, *Package)) {
	defer conn.Close()
	pack := readPackage(conn)
	if pack == nil {
		return
	}
	handle(conn, pack)
}

// Send sends the provided package to the specified address and returns the response
func Send(address string, pack *Package) *Package {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error while dialing:", err)
		return nil
	}
	defer conn.Close()
	_, err = conn.Write([]byte(SerializePackage(pack) + ENDBYTES))
	if err != nil {
		fmt.Println("Error while sending data:", err)
		return nil
	}
	ch := make(chan *Package)
	go func() {
		ch <- readPackage(conn)
	}()
	select {
	case res := <-ch:
		return res
	case <-time.After(WAITTIME):
		fmt.Println("Response timeout")
		return nil
	}
}

// SerializePackage serializes the provided package into a JSON string
func SerializePackage(pack *Package) string {
	jsonData, err := json.MarshalIndent(*pack, "", "\t")
	if err != nil {
		fmt.Println("Error while serializing package:", err)
		return ""
	}
	return string(jsonData)
}

// DeserializePackage deserializes the provided data string into a Package
func DeserializePackage(data string) *Package {
	var pack Package
	err := json.Unmarshal([]byte(data), &pack)
	if err != nil {
		fmt.Println("Error while deserializing package:", err)
		return nil
	}
	return &pack
}

// readPackage reads a package from the provided connection
func readPackage(conn net.Conn) *Package {
	var (
		data   string
		size   = uint64(0)
		buffer = make([]byte, BUFFSIZE)
	)
	for {
		length, err := conn.Read(buffer)
		if err != nil {
			fmt.Println("Error while reading data:", err)
			return nil
		}
		size += uint64(length)
		if size > DMAXSIZE {
			fmt.Println("Error: Data maximum size exceeded")
			return nil
		}
		data += string(buffer[:length])
		if strings.Contains(data, ENDBYTES) {
			data = strings.Split(data, ENDBYTES)[0]
			break
		}
	}
	return DeserializePackage(data)
}

// SendWakeUpMsgToAll sends a wake up message to all the specified addresses
func SendWakeUpMsgToAll(addresses []string) bool {
	var awakenodes uint64

	for _, address := range addresses {
		pack := &Package{
			Option: WAKEUP_MSG,
			Data:   "awake",
		}
		time.Sleep(time.Second / 5)
		fmt.Println("Sending wake up message to", address) // Debug statement
		response := Send(address, pack)
		if response != nil && response.Option == WAKEUP_MSG && response.Data == "awake" {
			fmt.Println("Received response from", address) // Debug statement
			awakenodes++
		} else {
			if response != nil {
				fmt.Println("Data is", response.Data) // Debug statement
			}
		}
	}

	return awakenodes > 1
}
