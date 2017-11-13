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

	dw := dateFromString( "18.01.2004")
	// праздничный или выходной день?
	rw, err :=ph.IsWeekend(dw)
	chkErr(err)
	fmt.Println("Result weekend:", rw)

	ds := dateFromString( "30.04.1999")
	// сокращённый предпраздничный?
	rs, err :=ph.IsShortDay(ds)
	chkErr(err)
	fmt.Println("Result shortday:", rs)

	// изменить интервал обновления кэша (по умолчанию 24 часа)
	ph.SetCacheTime(time.Second * 1)

	time.Sleep(time.Duration(time.Second * 2))

	d := dateFromString("01.01.2018")
	// всего рабочих дней
	wd, err := ph.WorkingDays(d)
	chkErr(err)
	// всего праздничных и выходных дней
	hd, err := ph.Holidays(d)
	chkErr(err)
	// количество рабочих часов при 40-часовой рабочей неделе
	h40, err := ph.WorkingHours40hWeek(d)
	chkErr(err)
	// количество рабочих часов при 36-часовой рабочей неделе
	h36, err := ph.WorkingHours36hWeek(d)
	chkErr(err)
	// количество рабочих часов при 24-часовой рабочей неделе
	h24, err := ph.WorkingHours24hWeek(d)
	chkErr(err)

	fmt.Printf("wd: %d, hd: %d, h40: %.01f, h36: %.01f, h24: %.01f\n", wd, hd, h40, h36, h24)
}

func dateFromString(str string) time.Time {
	d, err := time.Parse("02.01.2006", str)
	if err != nil {
		panic(err)
	}
	return d
}

func chkErr(err error) {
	if err != nil {
		fmt.Println("Return error:", err)
	}
}