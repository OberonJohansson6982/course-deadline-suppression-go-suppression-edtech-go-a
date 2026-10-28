package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: course-mailer email,name,course,YYYY-MM-DD")
		os.Exit(2)
	}
	l, err := parseLearner(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	c, err := newClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	suppressed, err := c.suppressionCheck(context.Background(), l.Email)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	decision := decideDelivery(l, suppressed, time.Now())
	fmt.Printf("learner=%s action=%s reason=%s\n", decision.Email, decision.Action, decision.Reason)
	if decision.Action == "send" {
		if err := c.sendDeadline(context.Background(), l); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("delivery=accepted")
	}
	if decision.Action == "suppress" {
		fmt.Println("report=hard-bounce-suppressed")
	}
	if decision.Action == "report" {
		fmt.Println("report=deadline-passed")
	}
}
