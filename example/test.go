package main

import (
	"github.com/supme/PublicHoliday"
	"time"
	"fmt"
)

func main() {
	ph := publicHoliday.New("you access token")

	fmt.Println("Manual update")
	err := ph.Update()
	chkErr(err)

	time.Sleep(time.Duration(time.Second * 2))

	dw, err := time.Parse("02.01.2006", "18.01.2004")
	if err != nil {
		panic(err)
	}
	rw, err :=ph.IsWeekend(dw)
	chkErr(err)
	fmt.Println("Result weekend:", rw)

	ds, err := time.Parse("02.01.2006", "30.04.1999")
	if err != nil {
		panic(err)
	}
	rs, err :=ph.IsShortDay(ds)
	chkErr(err)
	fmt.Println("Result shortday:", rs)

	ph.SetCacheTime(time.Second * 1)

	time.Sleep(time.Duration(time.Second * 2))

	d, err := time.Parse("02.01.2006", "01.01.2018")
	if err != nil {
		panic(err)
	}
	wd, err := ph.WorkingDays(d)
	chkErr(err)
	hd, err := ph.Holidays(d)
	chkErr(err)
	h40, err := ph.WorkingHours40hWeek(d)
	chkErr(err)
	h36, err := ph.WorkingHours36hWeek(d)
	chkErr(err)
	h24, err := ph.WorkingHours24hWeek(d)
	chkErr(err)

	fmt.Printf("wd: %d, hd: %d, h40: %.01f, h36: %.01f, h24: %.01f\n", wd, hd, h40, h36, h24)
}

func chkErr(err error) {
	if err != nil {
		fmt.Println("Return error:", err)
	}
}