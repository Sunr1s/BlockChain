package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	bc "github.com/Sunr1s/chain/blockchain"
	nt "github.com/Sunr1s/chain/network"
	"github.com/fatih/color"
)

const (
	LoadAddressPrefix  = "-loadaddr:"
	NewUserPrefix      = "-newuser:"
	LoadUserPrefix     = "-loaduser:"
	PanicMessagePrefix = "Initialization failed: "
)

var (
	infog     = color.New(color.FgGreen).PrintlnFunc()
	warng     = color.New(color.FgYellow).PrintlnFunc()
	errColorg = color.New(color.FgRed).PrintlnFunc()
	bcolorg   = color.New(color.BgCyan).PrintFunc()
)

func init() {
	if len(os.Args) < 2 {
		panic(PanicMessagePrefix + "No arguments provided")
	}

	var addrStr, userNewStr, userLoadStr string

	for _, arg := range os.Args[1:] {
		switch {
		case strings.HasPrefix(arg, LoadAddressPrefix):
			addrStr = strings.TrimPrefix(arg, LoadAddressPrefix)
		case strings.HasPrefix(arg, NewUserPrefix):
			userNewStr = strings.TrimPrefix(arg, NewUserPrefix)
			User = userNew(userNewStr)
		case strings.HasPrefix(arg, LoadUserPrefix):
			userLoadStr = strings.TrimPrefix(arg, LoadUserPrefix)
			User = userLoad(userLoadStr)
		}
	}

	if addrStr == "" || User == nil {
		panic(PanicMessagePrefix + "Invalid arguments")
	}

	err := json.Unmarshal([]byte(readFile(addrStr)), &Address)
	if err != nil {
		panic(PanicMessagePrefix + "Address loading failed")
	}
	if len(Address) == 0 {
		panic(PanicMessagePrefix + "No addresses loaded")
	}
}

// ASCIILogo returns HAMAHA word in ASCII
func ASCIILogo() string {
	return `

		██░ ██  ▄▄▄       ███▄ ▄███▓ ▄▄▄       ██░ ██  ▄▄▄
		▓██░ ██▒▒████▄    ▓██▒▀█▀ ██▒▒████▄    ▓██░ ██▒▒████▄
		▒██▀▀██░▒██  ▀█▄  ▓██    ▓██░▒██  ▀█▄  ▒██▀▀██░▒██  ▀█▄
		░▓█ ░██ ░██▄▄▄▄██ ▒██    ▒██ ░██▄▄▄▄██ ░▓█ ░██ ░██▄▄▄▄██
		░▓█▒░██▓ ▓█   ▓██▒▒██▒   ░██▒ ▓█   ▓██▒░▓█▒░██▓ ▓█   ▓██▒
		▒ ░░▒░▒ ▒▒   ▓▒█░░ ▒░   ░  ░ ▒▒   ▓▒█░ ▒ ░░▒░▒ ▒▒   ▓▒█░
		▒ ░▒░ ░  ▒   ▒▒ ░░  ░      ░  ▒   ▒▒ ░ ▒ ░▒░ ░  ▒   ▒▒ ░
		░  ░░ ░  ░   ▒   ░      ░     ░   ▒    ░  ░░ ░  ░   ▒
		░  ░  ░      ░  ░       ░         ░  ░ ░  ░  ░      ░  ░

		git 		github.com/Sunr1s/BlockChain

 `
}

func main() {
	infog(ASCIILogo())
	handleClientInput()
}

func handleClientInput() {
	makeTransaction([]string{"aaa", "3"})
	makeTransaction([]string{"aaa", "3"})

	for {
		message := inputString("> ")
		splitted := strings.Split(message, " ")

		switch splitted[0] {
		case "/exit":
			os.Exit(0)
		case "/user":
			handleUserCommand(splitted)
		case "/chain":
			handleChainCommand(splitted)
		default:
			errColorg("Undefined command")
		}
	}
}

func inputString(prompt string) string {
	fmt.Print(prompt)
	msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(msg)
}

// Command handlers
func handleUserCommand(args []string) {
	if len(args) < 2 {
		errColorg("Invalid command: user command expected")
		return
	}

	switch args[1] {
	case "address":
		printUserAddress()
	case "purse":
		printUserPurse()
	case "balance":
		printUserBalance()
	default:
		errColorg("Invalid command: unknown user command")
	}
}

func handleChainCommand(args []string) {
	if len(args) < 2 {
		errColorg("Invalid command: chain command expected")
		return
	}

	switch args[1] {
	case "print":
		printChain()
	case "tx":
		makeTransaction(args[2:])
	case "balance":
		printChainBalance(args[2:])
	default:
		errColorg("Invalid command: unknown chain command")
	}
}

// User command functions
func printUserAddress() {
	infog(fmt.Sprintf("Address: %s", User.Address()))
}

func printUserPurse() {
	infog(fmt.Sprintf("Purse: %s", User.Purse()))
}

func printUserBalance() {
	printBalance(User.Address())
}

// Chain command functions
func printChain() {
	for i := 0; ; i++ {
		response := nt.Send(Address[0], &nt.Package{
			Option: GET_BLOCK,
			Data:   fmt.Sprintf("%d", i),
		})
		if response == nil || response.Data == "" {
			break
		}
		fmt.Printf("[%d] => %s\n", i+1, response.Data)
	}
	fmt.Println()
}

func makeTransaction(args []string) {
	if len(args) != 2 {
		errColorg("Invalid transaction: receiver and amount expected")
		return
	}

	receiver := args[0]
	amount, err := strconv.Atoi(args[1])
	if err != nil {
		errColorg("Invalid transaction: invalid amount")
		return
	}

	for _, addr := range Address {
		response := nt.Send(addr, &nt.Package{
			Option: GET_LHASH,
		})
		if response == nil {
			continue
		}

		lastHash, _ := bc.Base64Decode(response.Data)
		tx, _ := bc.NewTransaction(User, lastHash, receiver, uint64(amount))

		serializedTx, _ := bc.SerializeTx(tx)
		response = nt.Send(addr, &nt.Package{
			Option: ADD_TRNSX,
			Data:   serializedTx,
		})

		if response == nil {
			continue
		}

		if response.Data == "ok" {
			infog(fmt.Sprintf("Transaction successful: %s", addr))
		} else {
			errColorg(fmt.Sprintf("Transaction failed: %s %s", addr, strings.Split(response.Data, "=")))
		}
	}
	fmt.Println()
}

func printChainBalance(args []string) {
	if len(args) != 1 {
		errColorg("Invalid command: user address expected")
		return
	}

	printBalance(args[0])
}

func printBalance(userAddr string) {
	for _, addr := range Address {
		response := nt.Send(addr, &nt.Package{
			Option: GET_BLNCE,
			Data:   userAddr,
		})
		if response == nil {
			continue
		}
		infog(fmt.Sprintf("Balance (%s): %s coins", addr, response.Data))
	}
	fmt.Println()
}
