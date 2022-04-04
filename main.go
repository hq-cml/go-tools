package main

import "fmt"

func main() {
	str := "人民币"
	for k, v := range str {
		fmt.Println(k, string(v))
	}
	fmt.Println("---------")
	str2 := []rune(str)
	for k, v := range str2 {
		fmt.Println(k, string(v))
	}
	fmt.Println("---------")
	str3 := []byte(str)
	for k, v := range str3 {
		fmt.Println(k, string(v))
	}
}
