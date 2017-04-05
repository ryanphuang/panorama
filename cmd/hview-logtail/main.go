package main

import (
	"github.com/hpcloud/tail"

	"fmt"
)

func main() {
	fmt.Println("vim-go")
	t, _ := tail.TailFile("/var/log/syslog", tail.Config{Follow: true})
	for line := range t.Lines {
		fmt.Println(line.Text)
	}
}
