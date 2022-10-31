package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	bc "./blockchain"
	nt "./network"
)

func init() {
	if len(os.Args) < 2 {
		panic("fail 1")
	}
	var (
		addrStr     = ""
		userNewStr  = ""
		userLoadStr = ""
	)
	var (
		addrExist     = false
		userNewExist  = false
		userLoadExist = false
	)
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case strings.HasPrefix(arg, "-loadaddr:"):
			addrStr = strings.Replace(arg, "-loadaddr:", "", 1)
			addrExist = true
		case strings.HasPrefix(arg, "-newuser:"):
			userNewStr = strings.Replace(arg, "-newuser:", "", 1)
			userNewExist = true
		case strings.HasPrefix(arg, "-loaduser:"):
			userLoadStr = strings.Replace(arg, "-loaduser:", "", 1)
			userLoadExist = true
		}
	}
	if !(userNewExist || userLoadExist || !addrExist) {
		panic("faild 333")
	}
	err := json.Unmarshal([]byte(readFile(addrStr)), &Address)
	if err != nil {
		panic("failed 3")
	}
	if len(Address) == 0 {
		panic("failed 4")
	}
	if userNewExist {
		User = userNew(userNewStr)
	}
	if userLoadExist {
		User = userLoad(userLoadStr)
	}
	if User == nil {
		panic("failed 5")
	}
}

func main() {
	handleClient()
}

func handleClient() {
	chainTX([]string{"tx", "aaa", "5"})
	chainTX([]string{"tx", "aaa", "5"})

	var (
		message string
		splited []string
	)
	for {
		message = inputString("> ")
		splited = strings.Split(message, " ")
		switch splited[0] {
		case "/exit":
			os.Exit(0)
		case "/user":
			if len(splited) < 2 {
				fmt.Println("len(user) <2")
				continue
			}
			switch splited[1] {
			case "address":
				userAddress()
			case "purse":
				userPurse()
			case "balance":
				userBalance()
			}
		case "/chain":
			if len(splited) < 2 {
				fmt.Println("len(user) < 2")
				continue
			}
			switch splited[1] {
			case "print":
				chainPrint()
			case "tx":
				chainTX(splited[1:])
			case "balance":
				chainBalance(splited[1:])
			}
		default:
			fmt.Println("undefined command\n")
		}
	}
}

func inputString(begin string) string {
	fmt.Printf(begin)
	msg, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.Replace(msg, "\n", "", 1)
}

func userAddress() {
	fmt.Println("Address:", User.Address(), "\n")
}

func userPurse() {
	fmt.Println("Purse:", User.Purse(), "\n")
}

func userBalance() {
	printBalance(User.Address())
}

func chainPrint() {
	for i := 0; ; i++ {
		res := nt.Send(Address[0], &nt.Package{
			Option: GET_BLOCK,
			Data:   fmt.Sprintf("%d", i),
		})
		if res == nil || res.Data == "" {
			break
		}
		fmt.Printf("[%d] => %s\n", i+1, res.Data)
	}
	fmt.Println()
}

func chainTX(splited []string) {
	if len(splited) != 3 {
		fmt.Println("failed: len(splited) != 3\n")
		return
	}
	num, err := strconv.Atoi(splited[2])
	if err != nil {
		fmt.Println("failed: strconv.Atoi(num)\n")
		return
	}
	for _, addr := range Address {
		res := nt.Send(addr, &nt.Package{
			Option: GET_LHASH,
		})
		if res == nil {
			continue
		}
		tx := bc.NewTransaction(User, bc.Base64Decode(res.Data), splited[1], uint64(num))
		res = nt.Send(addr, &nt.Package{
			Option: ADD_TRNSX,
			Data:   bc.SerializeTx(tx),
		})
		if res == nil {
			continue
		}
		if res.Data == "ok" {
			fmt.Printf("ok: (%s)\n", addr)
		} else {
			fmt.Printf("fail: (%s) (%s)\n", addr, strings.Split(res.Data, "="))
		}
	}
	fmt.Println()
}

func chainBalance(splited []string) {
	if len(splited) != 2 {
		fmt.Println("len(splited) != 2\n")
		return
	}
	printBalance(splited[1])
}

func printBalance(useraddr string) {
	for _, addr := range Address {
		res := nt.Send(addr, &nt.Package{
			Option: GET_BLNCE,
			Data:   useraddr,
		})
		if res == nil {
			continue
		}
		fmt.Printf("Balance (%s): %s coins\n", addr, res.Data)
	}
	fmt.Println()
}
