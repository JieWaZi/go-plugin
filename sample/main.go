package main

import "plugin"

func main() {
	p, err := plugin.Open("./go_plugin/sample/plugin/printnumber.so")
	if err != nil {
		panic(err)
	}

	printNumber, err := p.Lookup("PrintNumber1")
	if err != nil {
		panic(err)
	}
	printNumber.(func(int))(10)
}
